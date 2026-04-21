package notifications

import (
	"fmt"
	"log/slog"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
)

// Manager routes notifications to the configured backends (desktop, tmux, status file).
type Manager struct {
	cfg     config.NotificationConfig
	desktop *DesktopNotifier
	tmux    *TmuxNotifier
	status  *StatusWriter
}

// NewManager creates a new NotificationManager with the given configuration.
func NewManager(cfg config.NotificationConfig) *Manager {
	return &Manager{
		cfg:     cfg,
		desktop: &DesktopNotifier{},
		tmux:    &TmuxNotifier{},
		status:  &StatusWriter{},
	}
}

// Notify sends notifications for new messages through all enabled backends.
// It respects the notify_on setting: "none" suppresses all notifications,
// "unread" only notifies for unread messages, "all" notifies for everything.
func (m *Manager) Notify(msgs []db.Message) error {
	if m.cfg.NotifyOn == "none" || len(msgs) == 0 {
		return nil
	}

	// Filter messages based on notify_on setting.
	var relevant []db.Message
	for _, msg := range msgs {
		switch m.cfg.NotifyOn {
		case "unread":
			if !msg.IsRead {
				relevant = append(relevant, msg)
			}
		case "all":
			relevant = append(relevant, msg)
		}
	}

	if len(relevant) == 0 {
		return nil
	}

	// Build notification content.
	title, body := formatNotification(relevant)

	var errs []error

	// Desktop notification via beeep.
	if m.cfg.Desktop {
		if err := m.desktop.Notify(title, body); err != nil {
			slog.Warn("desktop notification failed", "error", err)
			errs = append(errs, fmt.Errorf("desktop: %w", err))
		}
	}

	// Terminal bell.
	if m.cfg.TerminalBell {
		// Print BEL character to trigger terminal bell.
		fmt.Print("\a")
	}

	// tmux window title.
	if m.cfg.TmuxTitle {
		unreadCount := len(relevant)
		m.tmux.SetTitle(unreadCount)
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}

	return nil
}

// WriteStatus writes the current status to the status file if enabled.
func (m *Manager) WriteStatus(database *db.DB) error {
	if !m.cfg.StatusFile {
		return nil
	}
	return m.status.Write(database)
}

// formatNotification builds a title and body string from a list of messages.
func formatNotification(msgs []db.Message) (title, body string) {
	count := len(msgs)
	if count == 1 {
		msg := msgs[0]
		title = fmt.Sprintf("New email from %s", msg.FromAddr)
		body = msg.Subject
	} else {
		title = fmt.Sprintf("%d new emails", count)
		// Show first few senders.
		body = ""
		limit := 3
		if count < limit {
			limit = count
		}
		for i := 0; i < limit; i++ {
			if i > 0 {
				body += ", "
			}
			body += msgs[i].FromAddr
		}
		if count > limit {
			body += fmt.Sprintf(" and %d more", count-limit)
		}
	}
	return title, body
}
