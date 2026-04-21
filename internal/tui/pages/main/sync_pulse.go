package mainpage

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/tui/msgs"
)

// SyncPulseCmd schedules SyncPulseMsg ticks for the initial-sync progress strip.
func SyncPulseCmd() tea.Cmd {
	return tea.Tick(140*time.Millisecond, func(time.Time) tea.Msg {
		return msgs.SyncPulseMsg{}
	})
}
