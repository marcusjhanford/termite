package daemon

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
	"github.com/termite-mail/termite/internal/engine"
	"github.com/termite-mail/termite/internal/notifications"
)

// Run starts the headless sync daemon loop. It opens the database, creates
// providers for all configured accounts, and runs incremental sync on the
// interval specified in cfg.General.CheckIntervalSeconds.
//
// Run blocks forever until an unrecoverable error occurs. It logs sync
// results and errors with slog and triggers notifications via the
// notifications manager when new mail arrives.
func Run(cfg *config.Config) error {
	database, err := db.Open()
	if err != nil {
		return fmt.Errorf("daemon: open database: %w", err)
	}
	defer database.Close()

	// Resolve accounts and their providers.
	accounts := make([]engine.Account, 0, len(cfg.Accounts))
	for _, acctCfg := range cfg.Accounts {
		acct, err := engine.NewAccount(acctCfg)
		if err != nil {
			slog.Error("daemon: failed to create account", "id", acctCfg.ID, "error", err)
			continue
		}

		// Ensure the account row exists in the DB.
		if err := database.InsertAccount(acctCfg.ID, acctCfg.Email, acctCfg.Provider, acctCfg.Name); err != nil {
			slog.Error("daemon: failed to insert account", "id", acctCfg.ID, "error", err)
			continue
		}

		accounts = append(accounts, acct)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("daemon: no accounts configured")
	}

	notifier := notifications.NewManager(cfg.Notifications)
	interval := time.Duration(cfg.General.CheckIntervalSeconds) * time.Second
	if interval < 10*time.Second {
		interval = 60 * time.Second
	}

	slog.Info("daemon: starting sync loop",
		"accounts", len(accounts),
		"interval", interval,
	)

	// Initial sync pass.
	for _, acct := range accounts {
		runSync(acct, database, cfg, notifier)
	}

	// Periodic sync loop.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, acct := range accounts {
			runSync(acct, database, cfg, notifier)
		}

		// Update the status file if enabled.
		if err := notifier.WriteStatus(database); err != nil {
			slog.Warn("daemon: failed to write status", "error", err)
		}
	}

	return nil
}

// runSync performs an incremental sync for a single account and handles
// notifications for new mail.
func runSync(acct engine.Account, database *db.DB, cfg *config.Config, notifier *notifications.Manager) {
	result := engine.IncrementalSync(acct, database, cfg.SplitInboxes)

	if result.Err != nil {
		slog.Error("daemon: sync failed",
			"account", result.AccountID,
			"error", result.Err,
		)
		return
	}

	slog.Info("daemon: sync complete",
		"account", result.AccountID,
		"new", result.NewCount,
	)

	// Notify on new mail.
	if result.NewCount > 0 {
		// Fetch the newest messages to pass to the notifier.
		msgs, err := database.GetThreadMessages(result.AccountID)
		if err != nil {
			slog.Warn("daemon: failed to fetch messages for notification",
				"account", result.AccountID,
				"error", err,
			)
			return
		}

		// Take only the most recent NewCount messages for notification.
		if len(msgs) > result.NewCount {
			msgs = msgs[len(msgs)-result.NewCount:]
		}

		if err := notifier.Notify(msgs); err != nil {
			slog.Warn("daemon: notification failed",
				"account", result.AccountID,
				"error", err,
			)
		}
	}
}
