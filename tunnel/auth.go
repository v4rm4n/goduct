package tunnel

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// BuildAuthMethods returns SSH auth methods based on what the user supplied.
// Priority order:
//  1. Explicit key file (--key)
//  2. SSH agent ($SSH_AUTH_SOCK) — used automatically if socket exists
//  3. Password (--password)
//
// At least one must succeed or the connection will fail.
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

	// 2. SSH agent — silent: only add if socket is available
	if agentMethod := authFromAgent(); agentMethod != nil {
		methods = append(methods, agentMethod)
	}

	// 3. If neither key nor agent, try default key locations
	if len(methods) == 0 {
		for _, defaultKey := range defaultKeyPaths() {
			method, err := authFromKeyFile(defaultKey)
			if err == nil {
				methods = append(methods, method)
				break // first one that loads wins
			}
		}
	}

	// 4. Password — last resort
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf(
			"no auth method available — provide --key, --password, or run ssh-agent",
		)
	}

	return methods, nil
}

// authFromKeyFile reads a PEM private key from disk and returns a Signer.
// Supports RSA, ECDSA, and Ed25519 keys.
func authFromKeyFile(path string) (ssh.AuthMethod, error) {
	// Expand ~ manually — os.ReadFile won't do it
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
// Returns nil (not an error) if no agent is available — it's optional.
func authFromAgent() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil // agent not reachable, skip silently
	}

	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

// defaultKeyPaths returns the standard SSH key locations to try when
// no explicit --key flag is given.
func defaultKeyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"), // preferred modern key
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}
}
