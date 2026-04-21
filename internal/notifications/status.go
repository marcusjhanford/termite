package notifications

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
)

// StatusWriter writes a JSON status file to ~/.termite/status.json.
// This allows external tools (tmux statusline scripts, polybar modules, etc.)
// to read Termite's current state without connecting to the process.
type StatusWriter struct{}

// statusJSON is the structure written to status.json.
type statusJSON struct {
	UnreadCount int    `json:"unread_count"`
	UpdatedAt   string `json:"updated_at"`
	Running     bool   `json:"running"`
}

// Write queries the database for the current unread count and writes
// the status to ~/.termite/status.json.
func (s *StatusWriter) Write(database *db.DB) error {
	unread, err := database.GetAllUnreadCount()
	if err != nil {
		return fmt.Errorf("status write: get unread count: %w", err)
	}

	status := statusJSON{
		UnreadCount: unread,
		UpdatedAt:   time.Now().Format(time.RFC3339),
		Running:     true,
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("status write: get data dir: %w", err)
	}

	statusPath := filepath.Join(dataDir, "status.json")

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("status write: marshal: %w", err)
	}

	// Write atomically: write to temp file then rename.
	tmpPath := statusPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("status write: write temp: %w", err)
	}

	if err := os.Rename(tmpPath, statusPath); err != nil {
		return fmt.Errorf("status write: rename: %w", err)
	}

	return nil
}

// Clear writes a status file indicating Termite is not running.
// Called on application exit.
func (s *StatusWriter) Clear() error {
	status := statusJSON{
		UnreadCount: 0,
		UpdatedAt:   time.Now().Format(time.RFC3339),
		Running:     false,
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("status clear: get data dir: %w", err)
	}

	statusPath := filepath.Join(dataDir, "status.json")

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("status clear: marshal: %w", err)
	}

	return os.WriteFile(statusPath, data, 0o644)
}
