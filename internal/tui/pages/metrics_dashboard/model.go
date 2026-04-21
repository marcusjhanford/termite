package metricsdashboard

import (
	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/metrics"
)

// Tab represents the active section in the metrics dashboard.
type Tab int

const (
	TabToday Tab = iota
	TabAllTime
	TabMilestones
	numTabs // sentinel
)

// MilestoneEntry is a display-friendly milestone for the view.
type MilestoneEntry struct {
	Icon       string
	Label      string
	Desc       string
	Unlocked   bool
	UnlockedAt string // formatted date, empty if locked
}

// Model is the metrics dashboard page model.
type Model struct {
	width  int
	height int

	activeTab Tab

	todaySummary metrics.DailySummary
	totals       metrics.Totals
	milestones   []MilestoneEntry
}

// New creates a new metrics dashboard model with stub data.
func New() Model {
	return Model{
		activeTab: TabToday,
		todaySummary: metrics.DailySummary{
			Date:       "2026-04-13",
			Cleared:    42,
			Sent:       7,
			InboxZeros: 2,
			TimeInApp:  1800, // 30 minutes
		},
		totals: metrics.Totals{
			TotalCleared:  1234,
			TotalSent:     256,
			TotalZeros:    89,
			LongestStreak: 14,
			CurrentStreak: 5,
		},
		milestones: stubMilestones(),
	}
}

// stubMilestones returns a set of sample milestone entries for display.
func stubMilestones() []MilestoneEntry {
	entries := make([]MilestoneEntry, 0, len(metrics.MilestoneDefinitions))
	// Mark a few as unlocked for demo purposes.
	unlocked := map[string]bool{
		"cleared_1":   true,
		"cleared_10":  true,
		"cleared_50":  true,
		"cleared_100": true,
		"sent_1":      true,
		"sent_10":     true,
		"zero_1":      true,
		"zero_7":      true,
		"streak_3":    true,
		"streak_7":    true,
	}

	for _, def := range metrics.MilestoneDefinitions {
		entry := MilestoneEntry{
			Icon:     def.Icon,
			Label:    def.Label,
			Desc:     def.Desc,
			Unlocked: unlocked[def.ID],
		}
		if entry.Unlocked {
			entry.UnlockedAt = "2026-04-10"
		}
		entries = append(entries, entry)
	}
	return entries
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
