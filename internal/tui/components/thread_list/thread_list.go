package threadlist

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/termite-mail/termite/internal/debuglog"
	"github.com/termite-mail/termite/internal/tui/linefmt"
)

// Each thread is rendered as a fixed-height "card" (two text rows) plus a
// one-line divider so every slot has the same screen height for scrolling.
const (
	threadCardScreenLines = 2
	threadDividerLines    = 1
	threadSlotScreenLines = threadCardScreenLines + threadDividerLines
	// threadDateCol reserves the right edge so relative times (e.g. "12mo") are not
	// clipped when clipOneLine trims the row to pane width.
	threadDateCol = 7
)

// ThreadItem represents a single thread entry.
type ThreadItem struct {
	ID            string
	Subject       string
	Sender        string
	Snippet       string
	Date          string
	MessageCount  int
	UnreadCount   int
	HasAttachment bool
}

// ThreadSelectedMsg is emitted when the user selects a thread.
// MarkRead is true when the user confirms with Enter (mark read + refresh inbox);
// false for j/k navigation (preview only).
type ThreadSelectedMsg struct {
	ThreadID string
	MarkRead bool
}

// Model is the middle-pane thread list component.
type Model struct {
	threads   []ThreadItem
	selected  int
	focused   bool
	width     int
	height    int
	offset    int    // scroll offset
	emptyHint string // shown when threads is empty (e.g. inbox has no unreads)
}

// New creates an empty thread list model.
func New() Model {
	return Model{}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input for the thread list.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if len(m.threads) > 0 && m.selected < len(m.threads)-1 {
				m.selected++
				m.ensureVisible()
				return m, m.emitSelected(false)
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.ensureVisible()
				return m, m.emitSelected(false)
			}
		case "enter":
			if len(m.threads) > 0 {
				// #region agent log
				debuglog.AgentLog("H5", "thread_list:enter", "thread list Enter → preview only (MarkRead false)", map[string]any{
					"threadID": m.threads[m.selected].ID, "markRead": false,
				})
				// #endregion
				return m, m.emitSelected(false)
			}
		}
	}

	return m, nil
}

func threadListTitleRendered() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1).
		Render("Threads")
}

// visibleThreadSlots returns how many thread slots (card + divider) fit under the title.
// Each card line is clipped to one row (see clipOneLine), so this stays aligned
// with the rendered layout.
func (m Model) visibleThreadSlots() int {
	titleH := lipgloss.Height(threadListTitleRendered())
	if titleH < 1 {
		titleH = 1
	}
	avail := m.height - titleH
	if avail < threadSlotScreenLines {
		avail = threadSlotScreenLines
	}
	n := avail / threadSlotScreenLines
	if n < 1 {
		return 1
	}
	return n
}

// clipOneLine forces a styled block to a single terminal row so lipgloss cannot
// wrap it inside a 2-row card (wrapping was breaking slot math and scrolling).
func (m Model) clipOneLine(block string) string {
	if m.width < 1 {
		return block
	}
	return lipgloss.NewStyle().Width(m.width).MaxHeight(1).Render(block)
}

// threadDivider renders a full-width separator row between cards.
func (m Model) threadDivider() string {
	if m.width < 1 {
		return ""
	}
	line := strings.Repeat("─", m.width)
	return lipgloss.NewStyle().
		Width(m.width).
		MaxHeight(1).
		Foreground(lipgloss.Color("#3a3a52")).
		Render(line)
}

// buildThreadCard renders one thread row (two clipped lines inside a Height(2) card).
func (m Model) buildThreadCard(i int) string {
	thread := m.threads[i]

	unreadIndicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6F61"))
	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))
	countBadgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))
	snippetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ABABAB"))

	indicator := "○"
	if thread.UnreadCount > 0 {
		indicator = unreadIndicatorStyle.Render("●")
	}
	attachment := ""
	if thread.HasAttachment {
		attachment = " 📎"
	}
	msgCount := ""
	if thread.MessageCount > 1 {
		msgCount = countBadgeStyle.Render(fmt.Sprintf(" [%d]", thread.MessageCount))
	}

	sender := linefmt.FormatJSONStringList(thread.Sender)
	sender = linefmt.CollapseWhitespace(sender)
	subject := linefmt.CollapseWhitespace(thread.Subject)

	// Budget for sender/subject: full width minus reserved date column (see clipOneLine).
	rest := m.width - 2 - lipgloss.Width(indicator) - threadDateCol -
		lipgloss.Width(msgCount) - lipgloss.Width(attachment) - 4
	if rest < 12 {
		rest = 12
	}
	leftBudget := rest * 2 / 5
	rightBudget := rest - leftBudget
	if leftBudget < 6 {
		leftBudget = 6
		rightBudget = rest - leftBudget
	}
	if rightBudget < 6 {
		rightBudget = 6
		leftBudget = rest - rightBudget
	}
	sender = linefmt.TruncateDisplayWidth(sender, leftBudget)
	subject = linefmt.TruncateDisplayWidth(subject, rightBudget)

	ds := dateStyle
	if i == m.selected {
		// Selected row uses a colored background; brighten the date for contrast.
		ds = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C8C8D0"))
	}
	date := ds.Width(threadDateCol).Align(lipgloss.Right).Render(thread.Date)

	leftMax := m.width - threadDateCol
	if leftMax < 1 {
		leftMax = 1
	}
	leftJoin := fmt.Sprintf(" %s %s — %s%s%s ", indicator, sender, subject, msgCount, attachment)
	leftBlock := lipgloss.NewStyle().MaxWidth(leftMax).Render(leftJoin)

	line1 := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, date)

	snippet := linefmt.CollapseWhitespace(thread.Snippet)
	snippet = linefmt.TruncateDisplayWidth(snippet, m.width-6)
	snippetLine := "     " + snippet

	if i == m.selected {
		l1 := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Render(line1)
		snColor := lipgloss.Color("#CCCCCC")
		if m.focused {
			snColor = lipgloss.Color("#E8E4F5")
		}
		l2 := lipgloss.NewStyle().
			Bold(false).
			Foreground(snColor).
			Render(snippetLine)
		row0 := m.clipOneLine(l1)
		row1 := m.clipOneLine(l2)
		inner := lipgloss.JoinVertical(lipgloss.Left, row0, row1)
		card := lipgloss.NewStyle().
			Width(m.width).
			Height(threadCardScreenLines).
			ColorWhitespace(true)
		if m.focused {
			card = card.Background(lipgloss.Color("#7D56F4"))
		} else {
			// Keep selection visible when focus is on another pane.
			card = card.Background(lipgloss.Color("#35354a"))
		}
		return card.Render(inner)
	}

	rowStyle := normalStyle
	snipStyle := snippetStyle
	if thread.UnreadCount > 0 {
		rowStyle = rowStyle.Bold(true).Foreground(lipgloss.Color("#D8D8EA"))
		snipStyle = snipStyle.Foreground(lipgloss.Color("#6B6B7E"))
	}
	inner := lipgloss.JoinVertical(lipgloss.Left,
		m.clipOneLine(rowStyle.Render(line1)),
		m.clipOneLine(snipStyle.Render(snippetLine)),
	)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(threadCardScreenLines).
		Render(inner)
}

// View renders the thread list.
func (m Model) View() string {
	if len(m.threads) == 0 {
		return m.emptyView()
	}

	titleRendered := threadListTitleRendered()
	slots := m.visibleThreadSlots()

	var rows []string
	rows = append(rows, titleRendered)

	end := m.offset + slots
	if end > len(m.threads) {
		end = len(m.threads)
	}
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.buildThreadCard(i))
		rows = append(rows, m.threadDivider())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// SetThreads replaces the current thread list.
func (m *Model) SetThreads(threads []ThreadItem) {
	m.threads = threads
	m.selected = 0
	m.offset = 0
}

// SetEmptyHint sets the copy shown when there are no threads (empty string restores default).
func (m *Model) SetEmptyHint(hint string) {
	m.emptyHint = hint
}

// SetFocused sets whether this component is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the available width and height.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SelectedThread returns the currently selected thread, or nil.
func (m Model) SelectedThread() *ThreadItem {
	if len(m.threads) == 0 {
		return nil
	}
	t := m.threads[m.selected]
	return &t
}

func (m Model) emptyView() string {
	text := "No threads"
	if m.emptyHint != "" {
		text = m.emptyHint
	}
	placeholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render(text)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(placeholder)
}

func (m Model) emitSelected(markRead bool) tea.Cmd {
	if len(m.threads) == 0 {
		return nil
	}
	id := m.threads[m.selected].ID
	return func() tea.Msg {
		return ThreadSelectedMsg{ThreadID: id, MarkRead: markRead}
	}
}

// ensureVisible adjusts the scroll offset so the selected item is visible.
func (m *Model) ensureVisible() {
	visibleItems := m.visibleThreadSlots()

	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleItems {
		m.offset = m.selected - visibleItems + 1
	}
}
