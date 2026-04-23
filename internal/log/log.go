// Package log provides centralised slog setup for Termite.
// When the TUI is active, logs must be written to a file so that daemon
// goroutines do not corrupt the terminal renderer.
package log

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/termite-mail/termite/internal/config"
)

// SetupFileLogger redirects the default slog logger to ~/.termite/termite.log.
// It should be called once, before any background goroutines start logging.
func SetupFileLogger() error {
	dir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("setup logger: %w", err)
	}

	path := filepath.Join(dir, "termite.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("setup logger: open log file: %w", err)
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
	return nil
}
