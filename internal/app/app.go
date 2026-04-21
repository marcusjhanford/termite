package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
	"github.com/termite-mail/termite/internal/metrics"
	"github.com/termite-mail/termite/internal/themes"
	"github.com/termite-mail/termite/internal/tui"
)

// App wires together all top-level application dependencies: configuration,
// database, theme management, and productivity metrics.
type App struct {
	Config       *config.Config
	DB           *db.DB
	ThemeManager *themes.ThemeManager
	Metrics      *metrics.MetricsTracker
}

// New creates a new App from the given configuration. It opens the database,
// initialises the theme manager with the configured theme, and creates a
// metrics tracker.
func New(cfg *config.Config) (*App, error) {
	database, err := db.Open()
	if err != nil {
		return nil, fmt.Errorf("app: open database: %w", err)
	}

	tm, err := themes.NewThemeManager()
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("app: create theme manager: %w", err)
	}

	// Apply the user's configured theme (fall back to default if it fails).
	if cfg.General.Theme != "" && cfg.General.Theme != "dark" {
		if applyErr := tm.Apply(cfg.General.Theme); applyErr != nil {
			// Log but don't fail — the default dark theme is already loaded.
			_ = applyErr
		}
	}

	tracker := metrics.NewTracker(database)

	return &App{
		Config:       cfg,
		DB:           database,
		ThemeManager: tm,
		Metrics:      tracker,
	}, nil
}

// Close releases all resources held by the App. It flushes the current
// metrics session and closes the database.
func (a *App) Close() error {
	// Flush the in-progress session metrics.
	if a.Metrics != nil {
		_ = a.Metrics.FlushSession()
	}

	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

// NewTUIModel creates and returns the root Bubble Tea model for the TUI,
// fully wired with the App's dependencies.
func (a *App) NewTUIModel() tea.Model {
	return tui.NewAppModel(a.Config, a.DB, a.ThemeManager, a.Metrics)
}
