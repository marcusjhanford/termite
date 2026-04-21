package mainpage

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// Pane width proportions: left=15%, middle=30%, right=55%.
const (
	leftPct   = 15
	middlePct = 30
	rightPct  = 55
)

// View implements tea.Model. It renders the three-pane layout with
// inbox list (left), thread list (middle), and message view (right),
// plus a title bar, status bar, and optional command bar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Title bar.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Background(lipgloss.Color("#1a1a2e"))

	titleDimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Background(lipgloss.Color("#1a1a2e"))

	titleText := titleStyle.Render(" termite ")
	titleSep := titleDimStyle.Render("│")
	var titleAccount string
	if len(m.cfg.Accounts) > 0 {
		titleAccount = titleDimStyle.Render(" " + m.cfg.Accounts[0].Email + " ")
	}
	titleLeft := titleText + titleSep + titleAccount
	titleGap := m.width - lipgloss.Width(titleLeft)
	if titleGap < 0 {
		titleGap = 0
	}
	titleBarBg := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a2e")).
		Width(m.width)
	titleBar := titleBarBg.Render(titleLeft)

	leftW := m.width * leftPct / 100
	midW := m.width * middlePct / 100
	rightW := m.width - leftW - midW // absorb rounding remainder

	// m.height already has status bar subtracted (set in WindowSizeMsg handler).
	// Reserve 1 for title bar, 1 for the optional sync strip, and 1 more when the command bar is visible.
	reserved := 1
	if m.syncStripVisible() {
		reserved++
	}
	if m.commandBar.IsActive() {
		reserved++
	}
	contentH := m.height - reserved
	if contentH < 1 {
		contentH = 1
	}

	// Base pane style with rounded border.
	baseBorder := lipgloss.RoundedBorder()
	baseStyle := lipgloss.NewStyle().
		Border(baseBorder).
		BorderForeground(lipgloss.Color("#555555"))

	focusedStyle := lipgloss.NewStyle().
		Border(baseBorder).
		BorderForeground(lipgloss.Color("#7D56F4"))

	// Build each pane using real component View() output.
	leftPane := m.renderPane("inbox",
		m.inboxList.View(), leftW-2, contentH-2,
		m.focus == FocusInboxList, baseStyle, focusedStyle,
	)
	midPane := m.renderPane("threads",
		m.threadList.View(), midW-2, contentH-2,
		m.focus == FocusThreadList, baseStyle, focusedStyle,
	)
	rightPane := m.renderPane("message",
		m.messageView.View(), rightW-2, contentH-2,
		m.focus == FocusMessageView, baseStyle, focusedStyle,
	)

	// Join the three panes horizontally.
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, midPane, rightPane)

	// Status bar spans the full width.
	statusLine := m.statusBar.View()

	// Command bar (only visible when active).
	commandLine := m.commandBar.View()

	var syncStrip string
	if m.syncStripVisible() {
		syncStrip = m.renderSyncProgressStrip()
	}

	rows := []string{titleBar}
	if syncStrip != "" {
		rows = append(rows, syncStrip)
	}
	rows = append(rows, panes, statusLine)
	if commandLine != "" {
		rows = append(rows, commandLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// ViewEmbeddedCompose renders the three-pane layout with the message reader on top
// of the right column and composeView below, sharing that column only (inbox and
// threads stay full height).
func (m Model) ViewEmbeddedCompose(composeView string) string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Background(lipgloss.Color("#1a1a2e"))

	titleDimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Background(lipgloss.Color("#1a1a2e"))

	titleText := titleStyle.Render(" termite ")
	titleSep := titleDimStyle.Render("│")
	var titleAccount string
	if len(m.cfg.Accounts) > 0 {
		titleAccount = titleDimStyle.Render(" " + m.cfg.Accounts[0].Email + " ")
	}
	titleLeft := titleText + titleSep + titleAccount
	titleBarBg := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a2e")).
		Width(m.width)
	titleBar := titleBarBg.Render(titleLeft)

	leftW := m.width * leftPct / 100
	midW := m.width * middlePct / 100
	rightW := m.width - leftW - midW

	reserved := 1
	if m.syncStripVisible() {
		reserved++
	}
	if m.commandBar.IsActive() {
		reserved++
	}
	contentH := m.height - reserved
	if contentH < 1 {
		contentH = 1
	}

	baseBorder := lipgloss.RoundedBorder()
	baseStyle := lipgloss.NewStyle().
		Border(baseBorder).
		BorderForeground(lipgloss.Color("#555555"))

	focusedStyle := lipgloss.NewStyle().
		Border(baseBorder).
		BorderForeground(lipgloss.Color("#7D56F4"))

	leftPane := m.renderPane("inbox",
		m.inboxList.View(), leftW-2, contentH-2,
		m.focus == FocusInboxList, baseStyle, focusedStyle,
	)
	midPane := m.renderPane("threads",
		m.threadList.View(), midW-2, contentH-2,
		m.focus == FocusThreadList, baseStyle, focusedStyle,
	)

	borderX, _ := paneBorder.GetFrameSize()
	innerW := rightW - 2 - borderX
	if innerW < 1 {
		innerW = 1
	}
	sepW := innerW
	sepLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Width(innerW).
		Render(strings.Repeat("─", sepW))

	msgBlock := lipgloss.NewStyle().Width(innerW).Render(m.messageView.View())
	rightStack := lipgloss.JoinVertical(lipgloss.Top, msgBlock, sepLine, composeView)
	rightPane := m.renderPane("message",
		rightStack, rightW-2, contentH-2,
		m.focus == FocusMessageView, baseStyle, focusedStyle,
	)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, midPane, rightPane)

	statusLine := m.statusBar.View()
	commandLine := m.commandBar.View()

	var syncStrip string
	if m.syncStripVisible() {
		syncStrip = m.renderSyncProgressStrip()
	}

	rows := []string{titleBar}
	if syncStrip != "" {
		rows = append(rows, syncStrip)
	}
	rows = append(rows, panes, statusLine)
	if commandLine != "" {
		rows = append(rows, commandLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderSyncProgressStrip is a single-line indeterminate bar plus account progress.
func (m Model) renderSyncProgressStrip() string {
	barW := m.width - 28
	if barW < 12 {
		barW = 12
	}
	if barW > 48 {
		barW = 48
	}
	seg := 3 + (m.syncPulse % (barW - 2))
	if seg < 2 {
		seg = 2
	}
	if seg > barW-2 {
		seg = barW - 2
	}
	var b strings.Builder
	for i := 0; i < barW; i++ {
		if i >= seg-2 && i < seg+2 {
			b.WriteString("█")
		} else {
			b.WriteString("░")
		}
	}
	label := "  Syncing mail"
	if m.bgSyncTotal > 1 {
		label += fmt.Sprintf("  (%d/%d)", m.bgSyncDone+1, m.bgSyncTotal)
	}
	line := label + "  " + b.String()
	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#1a1a2e")).
		Foreground(lipgloss.Color("#888888")).
		Render(line)
}

// renderPane renders a single pane with the given content, sizing, and focus state.
// Content is clipped to the bordered interior using Width/MaxHeight so wide lines
// cannot soft-wrap in the terminal (newline-only truncation was not enough).
func (m Model) renderPane(paneID, content string, w, h int, focused bool, base, focusStyle lipgloss.Style) string {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}

	style := base
	if focused {
		style = focusStyle
	}

	fx, fy := style.GetFrameSize()
	innerW := w - fx
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - fy
	if innerH < 1 {
		innerH = 1
	}

	inner := lipgloss.NewStyle().
		Width(innerW).
		MaxHeight(innerH).
		Render(content)

	return style.
		Width(w).
		Height(h).
		Render(inner)
}
