package mainpage

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
	"github.com/termite-mail/termite/internal/metrics"
	commandbar "github.com/termite-mail/termite/internal/tui/components/command_bar"
	inboxlist "github.com/termite-mail/termite/internal/tui/components/inbox_list"
	messageview "github.com/termite-mail/termite/internal/tui/components/message_view"
	statusbar "github.com/termite-mail/termite/internal/tui/components/status_bar"
	threadlist "github.com/termite-mail/termite/internal/tui/components/thread_list"
)

// FocusZone identifies which pane is focused.
type FocusZone int

const (
	FocusInboxList   FocusZone = iota // left pane
	FocusThreadList                   // middle pane
	FocusMessageView                  // right pane
)

const numZones = 3

// Model is the main three-pane layout model.
type Model struct {
	cfg      *config.Config
	database *db.DB
	tracker  *metrics.MetricsTracker

	width  int
	height int
	focus  FocusZone

	// Background initial sync progress (0 = no strip).
	bgSyncTotal int
	bgSyncDone  int
	syncPulse   int

	activeInboxID string

	inboxList   inboxlist.Model
	threadList  threadlist.Model
	messageView messageview.Model
	statusBar   statusbar.Model
	commandBar  commandbar.Model
}

// New creates a main page model wired to real components.
func New(cfg *config.Config, database *db.DB, tracker *metrics.MetricsTracker) Model {
	// Build inbox items from config split inboxes.
	items := make([]inboxlist.InboxItem, 0, len(cfg.SplitInboxes))
	for _, si := range cfg.SplitInboxes {
		items = append(items, inboxlist.InboxItem{
			ID:    si.ID,
			Label: si.Label,
		})
	}

	il := inboxlist.New(items)
	tl := threadlist.New()
	mv := messageview.New()
	sb := statusbar.New()
	cb := commandbar.New()

	// Determine active inbox.
	activeInbox := cfg.General.StartupInbox
	if activeInbox == "" {
		activeInbox = "primary"
	}
	if len(cfg.SplitInboxes) > 0 && !splitInboxIDExists(cfg.SplitInboxes, activeInbox) {
		if splitInboxIDExists(cfg.SplitInboxes, "primary") {
			activeInbox = "primary"
		} else {
			activeInbox = cfg.SplitInboxes[0].ID
		}
	}

	// Set initial status bar state.
	if len(cfg.Accounts) > 0 {
		sb.SetAccount(cfg.Accounts[0].Name)
	}
	sb.SetInbox(activeInbox)

	// Set initial focus on the thread list.
	tl.SetFocused(true)

	// Load initial unread count for status bar.
	if database != nil {
		unread, err := database.GetUnreadCount(activeInbox)
		if err != nil {
			slog.Warn("failed to load initial unread count", "err", err)
		} else {
			sb.SetUnread(unread)
		}
	}

	// Load initial metrics for status bar.
	if tracker != nil && cfg.Metrics.ShowInStatusbar {
		summary, err := tracker.TodaySummary()
		if err == nil {
			streak, _ := tracker.CurrentStreak()
			sb.SetMetrics(summary.Cleared, streak)
		}
	}

	return Model{
		cfg:           cfg,
		database:      database,
		tracker:       tracker,
		focus:         FocusThreadList,
		activeInboxID: activeInbox,
		inboxList:     il,
		threadList:    tl,
		messageView:   mv,
		statusBar:     sb,
		commandBar:    cb,
	}
}

func splitInboxIDExists(inboxes []config.SplitInboxConfig, id string) bool {
	for _, si := range inboxes {
		if si.ID == id {
			return true
		}
	}
	return false
}

// WithBackgroundSyncExpected returns a copy that shows a sync progress strip until
// bgSyncTotal initial sync jobs have finished (SyncDoneMsg or SyncErrorMsg each count).
func (m Model) WithBackgroundSyncExpected(n int) Model {
	if n <= 0 {
		return m
	}
	m.bgSyncTotal = n
	m.bgSyncDone = 0
	m.syncPulse = 0
	m.statusBar.SetSyncStatus("syncing…")
	return m
}

func (m Model) syncStripVisible() bool {
	return m.bgSyncTotal > 0 && m.bgSyncDone < m.bgSyncTotal
}

// advanceBackgroundSyncJob counts one finished background sync job (success or error).
func (m *Model) advanceBackgroundSyncJob() {
	if m.bgSyncTotal <= 0 {
		return
	}
	m.bgSyncDone++
	if m.bgSyncDone >= m.bgSyncTotal {
		m.bgSyncTotal = 0
		m.bgSyncDone = 0
	}
}

// CommandBarActive returns true if the command bar is currently active
// and consuming key events.
func (m Model) CommandBarActive() bool {
	return m.commandBar.IsActive()
}

// SetCommandNames passes the registered command names to the command bar
// for autocomplete.
func (m *Model) SetCommandNames(names []string) {
	m.commandBar.SetCommands(names)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
