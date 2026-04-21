package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "termite",
	Short: "A Superhuman-inspired TUI email client",
	Long: `Termite is a keyboard-first, terminal-native email client.
Built in Go with the Charm ecosystem. Your data stays yours.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.termite/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")

	// Set default command to 'run' when no subcommand is provided
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmd.RunE(cmd, args)
	}
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
