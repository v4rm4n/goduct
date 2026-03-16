package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/v4rm4n/goduct/tunnel"
)

var reverseCmd = &cobra.Command{
	Use:   "reverse [remotePort]:[localHost]:[localPort]",
	Short: "Expose a local port on the remote server (like ssh -R)",
	Args:  cobra.ExactArgs(1),
	Example: `  goduct reverse 9090:localhost:3000 --via user@jumphost
  goduct reverse 8080:localhost:8080 --via deploy@prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		via, _ := cmd.Flags().GetString("via")
		if via == "" {
			return fmt.Errorf("--via user@host is required")
		}

		cfg, err := tunnel.ParseReverseSpec(args[0], via)
		if err != nil {
			return err
		}

		cfg.KeyFile, _ = cmd.Flags().GetString("key")
		cfg.Password, _ = cmd.Flags().GetString("password")

		log.Printf("[goduct] reverse tunnel remote:%s -> %s:%s via %s",
			cfg.RemotePort, cfg.LocalHost, cfg.LocalPort, cfg.SSHHost)

		return tunnel.Reverse(cfg)
	},
}

func init() {
	reverseCmd.Flags().String("via", "", "SSH target in user@host[:port] format")
	reverseCmd.Flags().String("key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519, id_rsa)")
	reverseCmd.Flags().String("password", "", "SSH password (prefer keys or agent instead)")
	rootCmd.AddCommand(reverseCmd)
}
