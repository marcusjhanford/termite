package metricsdashboard

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// View renders the active section of the metrics dashboard.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Tab bar.
	sb.WriteString(m.renderTabBar())
	sb.WriteString("\n\n")

	// Active section content.
	switch m.activeTab {
	case TabToday:
		sb.WriteString(m.renderToday())
	case TabAllTime:
		sb.WriteString(m.renderAllTime())
	case TabMilestones:
		sb.WriteString(m.renderMilestones())
	}

	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("  Tab to switch sections  |  Esc to return"))

	return sb.String()
}

// renderTabBar renders the tab bar with the active tab highlighted.
func (m Model) renderTabBar() string {
	tabs := []string{"Today", "All Time", "Milestones"}
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 2)

	var rendered []string
	for i, tab := range tabs {
		if Tab(i) == m.activeTab {
			rendered = append(rendered, activeStyle.Render(tab))
		} else {
			rendered = append(rendered, inactiveStyle.Render(tab))
		}
	}

	return "  " + lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...)
}

// renderToday renders today's metrics summary with progress bars.
func (m Model) renderToday() string {
	var sb strings.Builder
	s := m.todaySummary

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("  Today's Summary"))
	sb.WriteString(fmt.Sprintf("  (%s)\n\n", s.Date))

	// Cleared.
	sb.WriteString(fmt.Sprintf("  Emails Cleared   %s  %d\n", progressBar(s.Cleared, 100, 20), s.Cleared))
	// Sent.
	sb.WriteString(fmt.Sprintf("  Emails Sent      %s  %d\n", progressBar(s.Sent, 50, 20), s.Sent))
	// Inbox Zeros.
	sb.WriteString(fmt.Sprintf("  Inbox Zeros      %s  %d\n", progressBar(s.InboxZeros, 5, 20), s.InboxZeros))
	// Time in App.
	minutes := s.TimeInApp / 60
	sb.WriteString(fmt.Sprintf("  Time in App      %s  %dm\n", progressBar(minutes, 120, 20), minutes))

	return sb.String()
}

// renderAllTime renders all-time totals with streak info.
func (m Model) renderAllTime() string {
	var sb strings.Builder
	t := m.totals

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("  All-Time Totals"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("  Total Cleared    %s  %d\n", progressBar(t.TotalCleared, 10000, 20), t.TotalCleared))
	sb.WriteString(fmt.Sprintf("  Total Sent       %s  %d\n", progressBar(t.TotalSent, 1000, 20), t.TotalSent))
	sb.WriteString(fmt.Sprintf("  Total Zeros      %s  %d\n", progressBar(t.TotalZeros, 100, 20), t.TotalZeros))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Current Streak   %d days\n", t.CurrentStreak))
	sb.WriteString(fmt.Sprintf("  Longest Streak   %d days\n", t.LongestStreak))

	return sb.String()
}

// renderMilestones renders the milestone grid showing locked/unlocked status.
func (m Model) renderMilestones() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("  Milestones"))
	sb.WriteString("\n\n")

	unlockedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	lockedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	cols := 3
	if m.width > 0 && m.width < 80 {
		cols = 2
	}
	if m.width > 0 && m.width < 50 {
		cols = 1
	}

	colWidth := 28
	if m.width > 0 {
		colWidth = (m.width - 4) / cols
		if colWidth < 20 {
			colWidth = 20
		}
	}

	for i, entry := range m.milestones {
		if i > 0 && i%cols == 0 {
			sb.WriteString("\n")
		}

		var cell string
		if entry.Unlocked {
			cell = unlockedStyle.Render(fmt.Sprintf(" %s %s", entry.Icon, entry.Label))
		} else {
			cell = lockedStyle.Render(fmt.Sprintf(" %s %s", "?", entry.Label))
		}

		// Pad to column width.
		cellRunes := []rune(cell)
		pad := colWidth - len(cellRunes)
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(cell)
		sb.WriteString(strings.Repeat(" ", pad))
	}

	// Summary.
	unlocked := 0
	for _, e := range m.milestones {
		if e.Unlocked {
			unlocked++
		}
	}
	sb.WriteString(fmt.Sprintf("\n\n  %d / %d milestones unlocked", unlocked, len(m.milestones)))

	return sb.String()
}

// progressBar renders a simple text progress bar.
// value is the current value, max is the target for a "full" bar,
// width is the bar width in characters.
func progressBar(value, max, width int) string {
	if max <= 0 {
		max = 1
	}
	filled := value * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))
}
