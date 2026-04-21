package statusbar

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// Model is the bottom status bar showing account, inbox, and metrics info.
type Model struct {
	account     string
	inbox       string
	unread      int
	syncStatus  string
	cleared     int
	streak      int
	showMetrics bool
	width       int
}

// New creates a default status bar model.
func New() Model {
	return Model{
		syncStatus: "idle",
	}
}

// View renders the status bar as a single horizontal line.
func (m Model) View() string {
	leftStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Background(lipgloss.Color("#333333")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#333333"))

	hintsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	// Build left section: [account] inbox · N unread · sync status
	var leftParts []string

	if m.account != "" {
		leftParts = append(leftParts, accentStyle.Render("["+m.account+"]"))
	}

	if m.inbox != "" {
		leftParts = append(leftParts, leftStyle.Render(m.inbox))
	}

	leftParts = append(leftParts,
		dimStyle.Render(fmt.Sprintf("%d unread", m.unread)))

	leftParts = append(leftParts,
		dimStyle.Render(m.syncStatus))

	left := strings.Join(leftParts, dimStyle.Render(" · "))

	// Build right section: metrics + key hints.
	var rightParts []string

	if m.showMetrics {
		metricsStr := fmt.Sprintf("cleared: %d  streak: %d", m.cleared, m.streak)
		rightParts = append(rightParts, dimStyle.Render(metricsStr))
	}

	hints := "j/k nav  tab pane  / search  : cmd  ? help  q quit"
	rightParts = append(rightParts, hintsStyle.Render(hints))

	right := strings.Join(rightParts, dimStyle.Render("  "))

	// Fill the gap between left and right.
	leftRendered := leftStyle.Render(left)
	rightRendered := right

	leftWidth := lipgloss.Width(leftRendered)
	rightWidth := lipgloss.Width(rightRendered)
	gap := m.width - leftWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	filler := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Render(strings.Repeat(" ", gap))

	bar := leftRendered + filler + rightRendered

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#333333")).
		Render(bar)
}

// SetAccount sets the displayed account name.
func (m *Model) SetAccount(name string) {
	m.account = name
}

// SetInbox sets the displayed inbox name.
func (m *Model) SetInbox(name string) {
	m.inbox = name
}

// SetUnread sets the unread message count.
func (m *Model) SetUnread(count int) {
	m.unread = count
}

// SetSyncStatus sets the sync status text.
func (m *Model) SetSyncStatus(status string) {
	m.syncStatus = status
}

// SetMetrics sets the cleared and streak values and enables metrics display.
func (m *Model) SetMetrics(cleared, streak int) {
	m.cleared = cleared
	m.streak = streak
	m.showMetrics = true
}

// SetWidth sets the rendering width.
func (m *Model) SetWidth(w int) {
	m.width = w
}
