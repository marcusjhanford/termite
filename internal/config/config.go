package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the root configuration for Termite.
type Config struct {
	General       GeneralConfig      `toml:"general"`
	Notifications NotificationConfig `toml:"notifications"`
	Metrics       MetricsConfig      `toml:"metrics"`
	Keybindings   KeybindingsConfig  `toml:"keybindings"`
	Accounts      []AccountConfig    `toml:"accounts"`
	SplitInboxes  []SplitInboxConfig `toml:"split_inboxes"`
}

// GeneralConfig holds general application settings.
type GeneralConfig struct {
	Theme                string `toml:"theme" validate:"required"`
	Editor               string `toml:"editor"`
	CheckIntervalSeconds int    `toml:"check_interval_seconds" validate:"min=10"`
	StartupInbox         string `toml:"startup_inbox"`
	ReduceMotion         bool   `toml:"reduce_motion"`
}

// NotificationConfig controls notification behavior.
type NotificationConfig struct {
	Desktop      bool   `toml:"desktop"`
	TerminalBell bool   `toml:"terminal_bell"`
	TmuxTitle    bool   `toml:"tmux_title"`
	StatusFile   bool   `toml:"status_file"`
	NotifyOn     string `toml:"notify_on" validate:"oneof=unread all none"`
}

// MetricsConfig controls the productivity metrics system.
type MetricsConfig struct {
	Enabled         bool `toml:"enabled"`
	ShowInStatusbar bool `toml:"show_in_statusbar"`
	ToastMilestones bool `toml:"toast_milestones"`
}

// KeybindingsConfig maps actions to key strings.
type KeybindingsConfig struct {
	Compose    string `toml:"compose"`
	Reply      string `toml:"reply"`
	ReplyAll   string `toml:"reply_all"`
	Forward    string `toml:"forward"`
	Archive    string `toml:"archive"`
	Delete     string `toml:"delete"`
	MarkRead   string `toml:"mark_read"`
	MarkUnread string `toml:"mark_unread"`
	Snooze     string `toml:"snooze"`
	Next       string `toml:"next"`
	Prev       string `toml:"prev"`
	Open       string `toml:"open"`
	Zero       string `toml:"zero"`
	Search     string `toml:"search"`
	Command    string `toml:"command"`
	Quit       string `toml:"quit"`
}

// AccountConfig defines an email account.
type AccountConfig struct {
	ID       string `toml:"id" validate:"required"`
	Name     string `toml:"name" validate:"required"`
	Email    string `toml:"email" validate:"required,email"`
	Provider string `toml:"provider" validate:"required,oneof=gmail outlook fastmail generic"`
}

// SplitInboxConfig defines a split inbox with filtering rules.
type SplitInboxConfig struct {
	ID       string      `toml:"id" validate:"required"`
	Label    string      `toml:"label" validate:"required"`
	Accounts []string    `toml:"accounts"`
	Rules    []InboxRule `toml:"rules"`
}

// InboxRule defines a single filtering rule for split inboxes.
type InboxRule struct {
	Field       string   `toml:"field"`
	Contains    []string `toml:"contains"`
	NotContains []string `toml:"not_contains"`
	Exists      *bool    `toml:"exists"`
	HeaderName  string   `toml:"header_name"`
}

// DataDir returns the path to Termite's data directory (~/.termite).
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	dir := filepath.Join(home, ".termite")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}
	return dir, nil
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads a TOML config from the given path. If path is empty,
// it tries the default location. Returns the parsed Config.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Save writes the config to the given path in TOML format.
func Save(cfg *Config, path string) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}
