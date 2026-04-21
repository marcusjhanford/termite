package mainpage

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	lipgloss "charm.land/lipgloss/v2"
	sqldb "github.com/termite-mail/termite/internal/db"
	commandbar "github.com/termite-mail/termite/internal/tui/components/command_bar"
	inboxlist "github.com/termite-mail/termite/internal/tui/components/inbox_list"
	threadlist "github.com/termite-mail/termite/internal/tui/components/thread_list"
	"github.com/termite-mail/termite/internal/tui/msgs"
)

// paneBorder matches view.renderPane borders — used to derive inner drawable height.
var paneBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

// Update implements tea.Model. It handles window resize, key events,
// and child-component messages for the three-pane main layout.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		// Reserve 1 row for status bar, 1 more if command bar is active.
		m.height = msg.Height - 1

		m.propagateSizes()
		return m, nil

	case tea.KeyPressMsg:
		// If command bar is active, route all keys to it.
		if m.commandBar.IsActive() {
			var cmd tea.Cmd
			m.commandBar, cmd = m.commandBar.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		if msg.String() == "esc" && m.searchResultsActive {
			m.searchResultsActive = false
			m.loadThreadsForInbox(m.activeInboxID)
			m.messageView.SetMessage("", "", "", "", "")
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.setFocus((m.focus + 1) % numZones)
			return m, nil
		case "shift+tab":
			m.setFocus((m.focus - 1 + numZones) % numZones)
			return m, nil
		case "/":
			m.commandBar.ActivateSearch()
			return m, nil
		case ":":
			m.commandBar.Activate()
			m.propagateSizes()
			return m, nil
		default:
			// Route to the focused component.
			return m.routeToFocused(msg)
		}

	case inboxlist.InboxSelectedMsg:
		m.activeInboxID = msg.InboxID
		m.loadThreadsForInbox(msg.InboxID)
		m.statusBar.SetInbox(msg.InboxID)
		m.refreshUnreadCount()
		return m, nil

	case threadlist.ThreadSelectedMsg:
		m.loadMessageForThread(msg.ThreadID)
		return m, nil

	case threadlist.InboxZeroMsg:
		// Emit InboxZeroMsg to the parent app model.
		accountID := ""
		if len(m.cfg.Accounts) > 0 {
			accountID = m.cfg.Accounts[0].ID
		}
		cmd := func() tea.Msg {
			return msgs.InboxZeroMsg{AccountID: accountID}
		}
		return m, cmd

	case commandbar.CommandMsg:
		m.propagateSizes()
		cmd := func() tea.Msg {
			return msgs.CommandMsg{Command: msg.Command}
		}
		return m, cmd

	case commandbar.SearchMsg:
		m.propagateSizes()
		cmd := m.searchThreads(msg.Query)
		return m, cmd

	case commandbar.CancelledMsg:
		m.propagateSizes()
		return m, nil

	case msgs.SyncPulseMsg:
		if !m.syncStripVisible() {
			return m, nil
		}
		m.syncPulse++
		m.propagateSizes()
		return m, SyncPulseCmd()

	case msgs.SyncDoneMsg:
		m.advanceBackgroundSyncJob()
		m.loadThreadsForInbox(m.activeInboxID)
		m.statusBar.SetSyncStatus("synced")
		m.refreshUnreadCount()
		m.refreshMetrics()
		m.propagateSizes()
		return m, nil

	case msgs.SyncErrorMsg:
		m.advanceBackgroundSyncJob()
		m.statusBar.SetSyncStatus("error")
		m.propagateSizes()
		return m, nil

	case msgs.NewMailMsg:
		m.loadThreadsForInbox(m.activeInboxID)
		m.statusBar.SetSyncStatus(fmt.Sprintf("+%d new", msg.Count))
		m.refreshUnreadCount()
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// setFocus updates the focus zone and propagates focus state to components.
func (m *Model) setFocus(zone FocusZone) {
	m.focus = zone
	m.inboxList.SetFocused(zone == FocusInboxList)
	m.threadList.SetFocused(zone == FocusThreadList)
	m.messageView.SetFocused(zone == FocusMessageView)
}

// routeToFocused sends a key message to whichever component is focused.
func (m Model) routeToFocused(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case FocusInboxList:
		m.inboxList, cmd = m.inboxList.Update(msg)
	case FocusThreadList:
		m.threadList, cmd = m.threadList.Update(msg)
	case FocusMessageView:
		m.messageView, cmd = m.messageView.Update(msg)
	}
	return m, cmd
}

// propagateSizes recalculates and distributes sizes to all child components.
func (m *Model) propagateSizes() {
	leftW := m.width * leftPct / 100
	midW := m.width * middlePct / 100
	rightW := m.width - leftW - midW

	// m.height already has status bar subtracted (set in WindowSizeMsg handler).
	// Subtract 1 for the title bar, 1 for the optional sync strip, and 1 more if command bar is active.
	contentH := m.height - 1
	if m.syncStripVisible() {
		contentH--
	}
	if m.commandBar.IsActive() {
		contentH--
	}
	if contentH < 1 {
		contentH = 1
	}
	// Outer height of each bordered pane (matches renderPane(..., contentH-2, ...)).
	paneOuterH := contentH - 2
	if paneOuterH < 3 {
		paneOuterH = 3
	}
	// Interior height: match lipgloss border frame (rounded border uses 2 vertical cells).
	borderX, borderY := paneBorder.GetFrameSize()
	paneInnerH := paneOuterH - borderY
	if paneInnerH < 1 {
		paneInnerH = 1
	}

	// Inner drawable width per pane: renderPane passes (colW-2) as bordered outer w,
	// then clips content to innerW = outer - borderX. Child SetSize must match innerW
	// so lipgloss row widths match the clipped region (avoids tight selection vs wide text).
	innerDrawable := func(outer int) int {
		w := outer - borderX
		if w < 1 {
			return 1
		}
		return w
	}
	inboxInnerW := innerDrawable(leftW - 2)
	threadInnerW := innerDrawable(midW - 2)
	msgInnerW := innerDrawable(rightW - 2)

	m.inboxList.SetSize(inboxInnerW, paneInnerH)
	m.threadList.SetSize(threadInnerW, paneInnerH)

	if m.embedCompose {
		// Reserve the bottom of the message column for inline compose.
		sep := 1
		msgPart := paneInnerH * 48 / 100
		if msgPart < 4 {
			msgPart = 4
		}
		composePart := paneInnerH - msgPart - sep
		if composePart < 6 {
			composePart = 6
			msgPart = paneInnerH - sep - composePart
			if msgPart < 3 {
				msgPart = 3
			}
		}
		m.messagePaneInnerH = msgPart
		m.composePaneInnerW = msgInnerW
		m.composePaneInnerH = composePart
		m.messageView.SetSize(msgInnerW, msgPart)
	} else {
		m.messagePaneInnerH = paneInnerH
		m.composePaneInnerW = 0
		m.composePaneInnerH = 0
		m.messageView.SetSize(msgInnerW, paneInnerH)
	}
	m.statusBar.SetWidth(m.width)
	m.commandBar.SetWidth(m.width)
}

// loadThreadsForInbox fetches threads from the database and updates the thread list.
func (m *Model) loadThreadsForInbox(inboxID string) {
	if m.database == nil {
		return
	}

	m.searchResultsActive = false

	dbThreads, err := m.database.GetThreads(inboxID, 100)
	if err != nil {
		slog.Warn("failed to load threads", "inbox", inboxID, "err", err)
		return
	}

	items := make([]threadlist.ThreadItem, 0, len(dbThreads))
	for _, t := range dbThreads {
		items = append(items, threadlist.ThreadItem{
			ID:            t.ID,
			Subject:       t.Subject,
			Sender:        t.Participants,
			Snippet:       t.Snippet,
			Date:          formatRelativeAge(t.LastMessageAt),
			MessageCount:  t.MessageCount,
			UnreadCount:   t.UnreadCount,
			HasAttachment: t.HasAttachment,
		})
	}

	m.threadList.SetThreads(items)
}

// loadMessageForThread fetches the latest message in a thread and displays it.
func (m *Model) loadMessageForThread(threadID string) {
	if m.database == nil {
		return
	}

	dbMsgs, err := m.database.GetThreadMessages(threadID)
	if err != nil {
		slog.Warn("failed to load messages", "thread", threadID, "err", err)
		return
	}

	if len(dbMsgs) == 0 {
		return
	}

	// Display the most recent message in the thread.
	latest := dbMsgs[len(dbMsgs)-1]
	m.messageView.SetMessage(
		latest.FromAddr,
		latest.ToAddrs,
		latest.Subject,
		formatMessageDateTime(latest.Date),
		latest.BodyText,
	)

	// Mark thread as read.
	if err := m.database.MarkThreadRead(threadID); err != nil {
		slog.Warn("failed to mark thread read", "thread", threadID, "err", err)
	}
}

// searchThreads performs FTS search and updates the thread list with results.
// On failure it returns a command that surfaces msgs.StatusMsg to the app root.
func (m *Model) searchThreads(query string) tea.Cmd {
	if m.database == nil {
		return nil
	}

	if strings.TrimSpace(query) != "" && sqldb.FormatFTS5MatchQuery(query) == "" {
		return func() tea.Msg {
			return msgs.StatusMsg{Text: "Search: enter a word or phrase to search"}
		}
	}

	searchMsgs, err := m.database.SearchMessages(query, 50)
	if err != nil {
		slog.Warn("search failed", "query", query, "err", err)
		errText := err.Error()
		return func() tea.Msg {
			return msgs.StatusMsg{Text: "Search failed: " + errText}
		}
	}

	// Deduplicate by thread ID and convert to ThreadItems.
	seen := make(map[string]bool)
	var items []threadlist.ThreadItem
	for _, msg := range searchMsgs {
		if seen[msg.ThreadID] {
			continue
		}
		seen[msg.ThreadID] = true
		items = append(items, threadlist.ThreadItem{
			ID:      msg.ThreadID,
			Subject: msg.Subject,
			Sender:  msg.FromAddr,
			Snippet: msg.BodyText,
			Date:    formatRelativeAge(msg.Date),
		})
	}

	m.threadList.SetThreads(items)
	m.searchResultsActive = true
	return nil
}

// refreshUnreadCount fetches unread count and updates the status bar.
func (m *Model) refreshUnreadCount() {
	if m.database == nil {
		return
	}
	count, err := m.database.GetUnreadCount(m.activeInboxID)
	if err != nil {
		slog.Warn("failed to get unread count", "err", err)
		return
	}
	m.statusBar.SetUnread(count)
}

// refreshMetrics updates the status bar metrics from the tracker.
func (m *Model) refreshMetrics() {
	if m.tracker == nil || !m.cfg.Metrics.ShowInStatusbar {
		return
	}
	summary, err := m.tracker.TodaySummary()
	if err != nil {
		slog.Warn("failed to get metrics summary", "err", err)
		return
	}
	streak, _ := m.tracker.CurrentStreak()
	m.statusBar.SetMetrics(summary.Cleared, streak)
}

// formatRelativeAge returns a compact “how long ago” string for thread rows (e.g. "45s", "3h", "1d", "2w").
func formatRelativeAge(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	s := int64(d.Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	case s < 24*3600:
		return fmt.Sprintf("%dh", s/3600)
	}
	days := s / (24 * 3600)
	switch {
	case days < 7:
		return fmt.Sprintf("%dd", days)
	case days < 30:
		return fmt.Sprintf("%dw", days/7)
	case days < 365:
		mo := days / 30
		if mo < 1 {
			mo = 1
		}
		return fmt.Sprintf("%dmo", mo)
	default:
		y := days / 365
		if y < 1 {
			y = 1
		}
		return fmt.Sprintf("%dy", y)
	}
}

// formatMessageDateTime formats a Unix instant as local calendar date and time for the message header.
func formatMessageDateTime(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0).In(time.Local)
	return t.Format("Mon, Jan 2, 2006 3:04 PM")
}
