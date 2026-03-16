package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goduct",
	Short: "SSH tunnels without the TTY pain",
	Long: `goduct is a dead-simple tunnel tool.
It speaks native SSH when possible, and HTTP tunneling as a fallback.

Examples:
  goduct forward 8080:localhost:80 --via user@host   # SSH local forward (uses agent or default keys)
  goduct forward 8080:localhost:80 --via user@host --key ~/.ssh/deploy_key
  goduct forward 8080:localhost:80 --via user@host --password s3cr3t
  goduct reverse 9090:localhost:90 --via user@host   # SSH remote forward
  goduct serve --port 2222                           # run goduct server (coming soon)
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
