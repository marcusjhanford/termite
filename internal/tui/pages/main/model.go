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

	// searchResultsActive is true after an FTS search until the user clears it (Esc) or reloads the inbox.
	searchResultsActive bool
	lastSearchQuery     string

	// lastUnreadTotal tracks the inbox unread count after the last load (-1 = not yet loaded).
	lastUnreadTotal int

	inboxList   inboxlist.Model
	threadList  threadlist.Model
	messageView messageview.Model
	statusBar   statusbar.Model
	commandBar  commandbar.Model

	// embedCompose: right column is split between message (top) and inline compose (bottom).
	embedCompose      bool
	messagePaneInnerH int
	composePaneInnerW int
	composePaneInnerH int
}

// New creates a main page model wired to real components.
func New(cfg *config.Config, database *db.DB, tracker *metrics.MetricsTracker) Model {
	il := inboxlist.New(nil)
	if database != nil {
		il = refreshInboxList(il, database)
	}

	tl := threadlist.New()
	mv := messageview.New()
	sb := statusbar.New()
	cb := commandbar.New()

	// Determine active inbox.
	activeInbox := cfg.General.StartupInbox
	if activeInbox == "" {
		activeInbox = "primary"
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
		cfg:             cfg,
		database:        database,
		tracker:         tracker,
		focus:           FocusThreadList,
		activeInboxID:   activeInbox,
		lastUnreadTotal: -1,
		inboxList:       il,
		threadList:      tl,
		messageView:     mv,
		statusBar:       sb,
		commandBar:      cb,
	}
}

// refreshInboxList rebuilds the inbox list items from the database with
// current unread counts. It preserves the current selection if possible.
func refreshInboxList(il inboxlist.Model, database *db.DB) inboxlist.Model {
	inboxes, err := database.ListSplitInboxes()
	if err != nil {
		slog.Warn("failed to load split inboxes", "err", err)
		return il
	}

	counts, err := database.GetUnreadCountByInbox()
	if err != nil {
		slog.Warn("failed to load unread counts", "err", err)
		counts = make(map[string]int)
	}

	items := make([]inboxlist.InboxItem, len(inboxes))
	for i, inbox := range inboxes {
		items[i] = inboxlist.InboxItem{
			ID:          inbox.ID,
			Label:       inbox.Label,
			UnreadCount: counts[inbox.ID],
		}
	}

	return inboxlist.New(items)
}

// RefreshInboxes updates the inbox list from the database and preserves the
// current selection. Returns the updated model.
func (m *Model) RefreshInboxes() {
	if m.database == nil {
		return
	}
	m.inboxList = refreshInboxList(m.inboxList, m.database)
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

// SelectedThreadID returns the focused thread ID, or empty if none.
func (m Model) SelectedThreadID() string {
	t := m.threadList.SelectedThread()
	if t == nil {
		return ""
	}
	return t.ID
}

// ActiveInboxID returns the currently active split inbox ID.
func (m Model) ActiveInboxID() string {
	return m.activeInboxID
}

// SetEmbedCompose toggles splitting the message column for inline reply/forward compose.
func (m *Model) SetEmbedCompose(v bool) {
	m.embedCompose = v
}

// ComposePaneInnerSize returns the last computed size for the compose stack in the right column.
func (m Model) ComposePaneInnerSize() (w, h int) {
	return m.composePaneInnerW, m.composePaneInnerH
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
