package tunnel

import (
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

// sshConnect dials the SSH server and returns an authenticated client.
// All auth logic lives in auth.go — this just uses the result.
func sshConnect(cfg *Config) (*ssh.Client, error) {
	authMethods, err := BuildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}

	clientCfg := &ssh.ClientConfig{
		User: cfg.SSHUser,
		Auth: authMethods,
		// TODO: replace with known_hosts verification before prod use
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", cfg.SSHHost, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", cfg.SSHHost, err)
	}

	return client, nil
}

// Forward opens a local listener and pipes each connection through SSH
// to remoteHost:remotePort on the other side.
// Equivalent to: ssh -N -L localPort:remoteHost:remotePort user@sshHost
func Forward(cfg *Config) error {
	client, err := sshConnect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	listenAddr := fmt.Sprintf("%s:%s", cfg.LocalHost, cfg.LocalPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("local listen on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] ready — listening on %s", listenAddr)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		// Each connection gets its own goroutine — no blocking
		go handleForward(client, localConn, cfg)
	}
}

// handleForward pipes one local connection → SSH tunnel → remote target.
func handleForward(client *ssh.Client, localConn net.Conn, cfg *Config) {
	defer localConn.Close()

	remoteAddr := fmt.Sprintf("%s:%s", cfg.RemoteHost, cfg.RemotePort)

	// Ask the SSH server to open a channel to remoteAddr
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[goduct] failed to reach %s via SSH: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	log.Printf("[goduct] new connection: local -> %s", remoteAddr)

	// Bidirectional pipe: copy in both directions simultaneously
	pipe(localConn, remoteConn)
}

// Reverse asks the SSH server to bind a port on its end, and for every
// incoming connection there it dials back to localHost:localPort on our side.
// Equivalent to: ssh -N -R remotePort:localHost:localPort user@sshHost
func Reverse(cfg *Config) error {
	client, err := sshConnect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	// Ask the SSH *server* to listen — note: client.Listen not net.Listen
	remoteAddr := fmt.Sprintf("0.0.0.0:%s", cfg.RemotePort)
	listener, err := client.Listen("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("remote listen on %s: %w", remoteAddr, err)
	}
	defer listener.Close()

	log.Printf("[goduct] ready — remote port %s -> %s:%s",
		cfg.RemotePort, cfg.LocalHost, cfg.LocalPort)

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		go handleReverse(remoteConn, cfg)
	}
}

// handleReverse pipes one remote SSH connection → local target.
func handleReverse(remoteConn net.Conn, cfg *Config) {
	defer remoteConn.Close()

	localAddr := fmt.Sprintf("%s:%s", cfg.LocalHost, cfg.LocalPort)

	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		log.Printf("[goduct] failed to reach local %s: %v", localAddr, err)
		return
	}
	defer localConn.Close()

	log.Printf("[goduct] new reverse connection: remote -> %s", localAddr)

	pipe(remoteConn, localConn)
}

// pipe copies data between two connections in both directions.
// Blocks until either side closes.
func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(a, b) //nolint:errcheck
		done <- struct{}{}
	}()
	go func() {
		io.Copy(b, a) //nolint:errcheck
		done <- struct{}{}
	}()

	// Wait for either direction to finish, then close both
	<-done
}
