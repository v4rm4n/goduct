// Package tunnel handles all port-forwarding logic.
// Currently supports native SSH. HTTP tunneling planned.
package tunnel

import (
	"fmt"
	"strings"
)

// Config holds everything needed to open a forward tunnel.
// Think: ssh -L localPort:remoteHost:remotePort user@sshHost
type Config struct {
	SSHUser    string
	SSHHost    string // host:port, port defaults to 22
	LocalHost  string // usually 127.0.0.1
	LocalPort  string
	RemoteHost string
	RemotePort string

	// Auth — tried in this order: KeyFile → SSHAgent → Password
	KeyFile  string // path to private key, e.g. ~/.ssh/id_rsa
	Password string // fallback password auth
}

// ParseForwardSpec parses "localPort:remoteHost:remotePort" + "user@host[:port]"
// e.g. "8080:localhost:80" --via "alice@jumphost"
func ParseForwardSpec(spec, via string) (*Config, error) {
	cfg, err := parseVia(via)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(spec, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("forward spec must be localPort:remoteHost:remotePort, got %q", spec)
	}

	cfg.LocalHost = "127.0.0.1"
	cfg.LocalPort = parts[0]
	cfg.RemoteHost = parts[1]
	cfg.RemotePort = parts[2]

	return cfg, nil
}

// ParseReverseSpec parses "remotePort:localHost:localPort" + "user@host[:port]"
// e.g. "9090:localhost:3000" --via "alice@prod"
func ParseReverseSpec(spec, via string) (*Config, error) {
	cfg, err := parseVia(via)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(spec, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("reverse spec must be remotePort:localHost:localPort, got %q", spec)
	}

	cfg.RemotePort = parts[0]
	cfg.LocalHost = parts[1]
	cfg.LocalPort = parts[2]

	return cfg, nil
}

// parseVia splits "user@host" or "user@host:port"
func parseVia(via string) (*Config, error) {
	cfg := &Config{}

	atIdx := strings.Index(via, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("--via must be user@host[:port], got %q", via)
	}

	cfg.SSHUser = via[:atIdx]
	hostpart := via[atIdx+1:]

	// if no port, default to 22
	if strings.Contains(hostpart, ":") {
		cfg.SSHHost = hostpart
	} else {
		cfg.SSHHost = hostpart + ":22"
	}

	return cfg, nil
}
