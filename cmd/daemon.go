package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run headless background sync",
	Long:  "Runs the Termite sync daemon in the foreground. Use 'termite install-daemon' to set up as a system service.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		slog.Info("starting termite daemon")
		return daemon.Run(cfg)
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
