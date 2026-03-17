// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/v4rm4n/goduct/tunnel"
)

var rootCmd = &cobra.Command{
	Use:   "goduct [source] [-f- or -r-] [destination] [user@host]",
	Short: "SSH tunnels without the TTY pain",
	Long: `goduct is a dead-simple tunnel tool using intuitive connector syntax.

Local Forwarding (-f-): Listen locally, connect to remote via SSH
  goduct eth0:8080 -f- localhost:80 admin@prod
  goduct Wi-Fi:3000 -f- db.internal:5432 user@bastion

Remote Forwarding (-r-): SSH server listens, connects back to your local machine
  goduct 0.0.0.0:9090 -r- localhost:3000 dev@staging
  goduct localhost:8080 -r- Ethernet:80 dev@staging`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceStr := args[0]
		connector := args[1]
		destStr := args[2]
		viaStr := args[3]

		cfg, err := tunnel.ParseVia(viaStr)
		if err != nil {
			return err
		}

		cfg.KeyFile, _ = cmd.Flags().GetString("key")

		switch connector {
		case "FWD": // Intercepted '-f-'
			bindAddr, err := tunnel.ResolveHostPort(sourceStr)
			if err != nil {
				return fmt.Errorf("source error: %w", err)
			}
			cfg.BindAddr = bindAddr
			cfg.TargetAddr = destStr
			return tunnel.Forward(cfg)

		case "REV": // Intercepted '-r-'
			cfg.BindAddr = sourceStr
			targetAddr, err := tunnel.ResolveHostPort(destStr)
			if err != nil {
				return fmt.Errorf("destination error: %w", err)
			}
			cfg.TargetAddr = targetAddr
			return tunnel.Reverse(cfg)

		default:
			return fmt.Errorf("invalid direction %q: must be '-f-' (forward) or '-r-' (reverse)", args[1]) // We use args[1] so the error shows what the user actually typed
		}
	},
}

func Execute() {
	// Hack to prevent Cobra from thinking -f- and -r- are flags
	// We swap them out for internal strings before Cobra parses them.
	for i, arg := range os.Args {
		if arg == "-f-" {
			os.Args[i] = "FWD"
		} else if arg == "-r-" {
			os.Args[i] = "REV"
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String("key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519, id_rsa)")
}
