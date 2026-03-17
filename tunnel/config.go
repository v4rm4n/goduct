// tunnel/config.go
package tunnel

import (
	"fmt"
	"strings"
)

type Config struct {
	SSHUser    string
	SSHHost    string
	BindAddr   string // Local listen address for Forward, Remote listen address for Reverse
	TargetAddr string // Remote target for Forward, Local target for Reverse

	KeyFile  string
	Password string
}

// ParseVia splits "user@host" or "user@host:port"
func ParseVia(via string) (*Config, error) {
	cfg := &Config{}

	atIdx := strings.Index(via, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("SSH target must be user@host[:port], got %q", via)
	}

	cfg.SSHUser = via[:atIdx]
	hostpart := via[atIdx+1:]

	if strings.Contains(hostpart, ":") {
		cfg.SSHHost = hostpart
	} else {
		cfg.SSHHost = hostpart + ":22"
	}

	return cfg, nil
}
