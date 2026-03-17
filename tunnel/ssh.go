// tunnel/ssh.go
package tunnel

import (
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

func sshConnect(cfg *Config) (*ssh.Client, error) {
	authMethods, err := BuildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}

	clientCfg := &ssh.ClientConfig{
		User:            cfg.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", cfg.SSHHost, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", cfg.SSHHost, err)
	}

	return client, nil
}

// Forward opens a local listener and pipes connections to the remote target.
func Forward(cfg *Config) error {
	client, err := sshConnect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("local listen on %s: %w", cfg.BindAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] Forward: listening locally on %s -> remote %s via %s", cfg.BindAddr, cfg.TargetAddr, cfg.SSHHost)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		go handleForward(client, localConn, cfg.TargetAddr)
	}
}

func handleForward(client *ssh.Client, localConn net.Conn, targetAddr string) {
	defer localConn.Close()

	remoteConn, err := client.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("[goduct] failed to reach %s via SSH: %v", targetAddr, err)
		return
	}
	defer remoteConn.Close()

	pipe(localConn, remoteConn)
}

// Reverse binds a port on the remote SSH server and dials back to a local target.
func Reverse(cfg *Config) error {
	client, err := sshConnect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	listener, err := client.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("remote listen on %s: %w", cfg.BindAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] Reverse: remote server listening on %s -> local %s via %s", cfg.BindAddr, cfg.TargetAddr, cfg.SSHHost)

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
		log.Printf("[goduct] failed to reach local %s: %v", targetAddr, err)
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
