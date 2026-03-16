package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/v4rm4n/goduct/tunnel"
)

var forwardCmd = &cobra.Command{
	Use:   "forward [localPort]:[remoteHost]:[remotePort]",
	Short: "Forward a local port to a remote destination (like ssh -L)",
	Args:  cobra.ExactArgs(1),
	Example: `  goduct forward 8080:localhost:80 --via user@jumphost
  goduct forward 5432:db.internal:5432 --via admin@bastion`,
	RunE: func(cmd *cobra.Command, args []string) error {
		via, _ := cmd.Flags().GetString("via")
		if via == "" {
			return fmt.Errorf("--via user@host is required")
		}

		cfg, err := tunnel.ParseForwardSpec(args[0], via)
		if err != nil {
			return err
		}

		cfg.KeyFile, _ = cmd.Flags().GetString("key")
		cfg.Password, _ = cmd.Flags().GetString("password")

		log.Printf("[goduct] forwarding 127.0.0.1:%s -> %s:%s via %s",
			cfg.LocalPort, cfg.RemoteHost, cfg.RemotePort, cfg.SSHHost)

		return tunnel.Forward(cfg)
	},
}

func init() {
	forwardCmd.Flags().String("via", "", "SSH target in user@host[:port] format")
	forwardCmd.Flags().String("key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519, id_rsa)")
	forwardCmd.Flags().String("password", "", "SSH password (prefer keys or agent instead)")
	rootCmd.AddCommand(forwardCmd)
}
