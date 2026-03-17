// tunnel/auth.go
package tunnel

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

// BuildAuthMethods returns SSH auth methods.
// Priority order: Explicit key -> SSH agent -> Default keys -> Interactive Password
func BuildAuthMethods(cfg *Config) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// 1. Explicit key file
	if cfg.KeyFile != "" {
		method, err := authFromKeyFile(cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("key file %q: %w", cfg.KeyFile, err)
		}
		methods = append(methods, method)
	}

	// 2. SSH agent
	if agentMethod := authFromAgent(); agentMethod != nil {
		methods = append(methods, agentMethod)
	}

	// 3. Default key locations
	if len(methods) == 0 {
		for _, defaultKey := range defaultKeyPaths() {
			method, err := authFromKeyFile(defaultKey)
			if err == nil {
				methods = append(methods, method)
				break
			}
		}
	}

	// 4. Interactive Password fallback
	// This callback is only executed by the SSH client if keys/agent fail
	// and the server explicitly requests a password.
	interactivePassword := ssh.PasswordCallback(func() (secret string, err error) {
		fmt.Fprintf(os.Stderr, "Password for %s@%s: ", cfg.SSHUser, cfg.SSHHost)

		// ReadPassword hides input natively
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // Print a newline after they hit enter

		if err != nil {
			return "", err
		}
		return string(bytePassword), nil
	})

	methods = append(methods, interactivePassword)

	return methods, nil
}

// authFromKeyFile reads a PEM private key from disk and returns a Signer.
func authFromKeyFile(path string) (ssh.AuthMethod, error) {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, path[2:])
	}

	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// authFromAgent connects to the running SSH agent via $SSH_AUTH_SOCK.
func authFromAgent() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}

	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

// defaultKeyPaths returns the standard SSH key locations to try.
func defaultKeyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}
}
