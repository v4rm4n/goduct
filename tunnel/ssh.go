// tunnel/ssh.go
package tunnel

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

const reconnectDelay = 3 * time.Second

func sshConnect(cfg *Config) (*ssh.Client, error) {
	authMethods, err := BuildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}

	clientCfg := &ssh.ClientConfig{
		User:            cfg.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second, // Prevent hanging forever on dead hosts
	}

	client, err := ssh.Dial("tcp", cfg.SSHHost, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", cfg.SSHHost, err)
	}

	return client, nil
}

// Forward opens a local listener and pipes connections to the remote target.
// It automatically reconnects if the SSH session drops.
func Forward(cfg *Config) error {
	// 1. Start local listener outside the reconnect loop so the port stays open
	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("local listen on %s: %w", cfg.BindAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] Forward: listening locally on %s -> remote %s via %s", cfg.BindAddr, cfg.TargetAddr, cfg.SSHHost)

	// Channel to pipe local connections to the active SSH session
	conns := make(chan net.Conn)

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("[goduct] Local accept error: %v", err)
				continue
			}
			conns <- localConn
		}
	}()

	// 2. Outer reconnect loop
	for {
		client, err := sshConnect(cfg)
		if err != nil {
			log.Printf("[goduct] SSH connection failed: %v. Retrying in %v...", err, reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}
		log.Printf("[goduct] SSH session established to %s", cfg.SSHHost)

		// Start the jittered keep-alive routine
		go keepAlive(client)

		// Monitor for SSH session drops
		drop := make(chan error, 1)
		go func() {
			drop <- client.Wait()
		}()

	proxyLoop:
		for {
			select {
			case err := <-drop:
				log.Printf("[goduct] SSH session dropped: %v. Reconnecting...", err)
				client.Close()
				break proxyLoop // Break to the outer loop to reconnect
			case localConn := <-conns:
				go handleForward(client, localConn, cfg.TargetAddr)
			}
		}
		time.Sleep(reconnectDelay)
	}
}

func handleForward(client *ssh.Client, localConn net.Conn, targetAddr string) {
	defer localConn.Close()

	remoteConn, err := client.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("[goduct] Failed to reach %s via SSH: %v", targetAddr, err)
		return
	}
	defer remoteConn.Close()

	pipe(localConn, remoteConn)
}

// Reverse binds a port on the remote SSH server and dials back to a local target.
// It automatically reconnects if the SSH session drops.
func Reverse(cfg *Config) error {
	for {
		client, err := sshConnect(cfg)
		if err != nil {
			log.Printf("[goduct] SSH connection failed: %v. Retrying in %v...", err, reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}
		log.Printf("[goduct] SSH session established to %s", cfg.SSHHost)

		err = serveReverse(client, cfg)
		log.Printf("[goduct] Reverse tunnel dropped: %v. Reconnecting in %v...", err, reconnectDelay)

		client.Close()
		time.Sleep(reconnectDelay)
	}
}

func serveReverse(client *ssh.Client, cfg *Config) error {
	listener, err := client.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("remote listen on %s: %w", cfg.BindAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] Reverse: remote server listening on %s -> local %s via %s", cfg.BindAddr, cfg.TargetAddr, cfg.SSHHost)

	// If Accept fails, the underlying SSH connection was likely closed
	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		go handleReverse(remoteConn, cfg.TargetAddr)
	}
}

func handleReverse(remoteConn net.Conn, targetAddr string) {
	defer remoteConn.Close()

	localConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("[goduct] Failed to reach local %s: %v", targetAddr, err)
		return
	}
	defer localConn.Close()

	pipe(remoteConn, localConn)
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(a, b)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(b, a)
		done <- struct{}{}
	}()
	<-done
}

// keepAlive runs in a goroutine and sends jittered SSH global requests
// to prevent firewall timeouts without creating a predictable beacon signature.
func keepAlive(client *ssh.Client) {
	for {
		// Randomize the interval between 15 and 45 seconds
		jitter := time.Duration(15+rand.Intn(30)) * time.Second
		time.Sleep(jitter)

		// Send an SSH global request. The payload is encrypted.
		// wantReply=true forces the server to respond, verifying the connection is alive.
		_, _, err := client.SendRequest("keepalive@goduct", true, nil)
		if err != nil {
			// If this fails, the connection is dead.
			// client.Wait() in your main loop will catch this and trigger a reconnect.
			log.Printf("[goduct] Jittered keep-alive failed: %v", err)
			return
		}
	}
}
