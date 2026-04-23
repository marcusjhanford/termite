package cmd

import (
	"fmt"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/termite-mail/termite/internal/app"
	"github.com/termite-mail/termite/internal/config"
	termitelog "github.com/termite-mail/termite/internal/log"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch the Termite TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := termitelog.SetupFileLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "termite: failed to setup logger: %v\n", err)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			slog.Info("no config found, creating default config")
			cfg = config.Default()
			// Write the default config so the user has a file to edit
			if saveErr := config.Save(cfg, cfgFile); saveErr != nil {
				slog.Warn("could not write default config", "err", saveErr)
			} else {
				path, _ := config.DefaultConfigPath()
				if cfgFile != "" {
					path = cfgFile
				}
				slog.Info("wrote default config", "path", path)
			}
		}

		application, err := app.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize app: %w", err)
		}
		defer application.Close()

		model := application.NewTUIModel()
		p := tea.NewProgram(model)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "termite: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
