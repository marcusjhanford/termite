package metrics

import "time"

// MilestoneDef defines a milestone that can be unlocked by the user.
type MilestoneDef struct {
	ID        string
	Category  string // "cleared" | "sent" | "streak" | "zero"
	Threshold int
	Icon      string
	Label     string
	Desc      string
}

// Milestone represents an unlocked milestone with its unlock timestamp.
type Milestone struct {
	MilestoneDef
	UnlockedAt time.Time
}

// MilestoneDefinitions contains all milestone definitions from the spec.
var MilestoneDefinitions = []MilestoneDef{
	// --- Emails cleared ---
	{ID: "cleared_1", Category: "cleared", Threshold: 1, Icon: "✦", Label: "First Clear", Desc: "Archived or deleted your first email"},
	{ID: "cleared_10", Category: "cleared", Threshold: 10, Icon: "◆", Label: "Getting Started", Desc: "10 emails cleared"},
	{ID: "cleared_50", Category: "cleared", Threshold: 50, Icon: "▲", Label: "Momentum", Desc: "50 emails cleared"},
	{ID: "cleared_100", Category: "cleared", Threshold: 100, Icon: "●", Label: "Century", Desc: "100 emails cleared"},
	{ID: "cleared_500", Category: "cleared", Threshold: 500, Icon: "★", Label: "Five Hundred", Desc: "500 emails cleared"},
	{ID: "cleared_1000", Category: "cleared", Threshold: 1000, Icon: "✸", Label: "The Archivist", Desc: "1,000 emails cleared. Your inbox fears you."},
	{ID: "cleared_5000", Category: "cleared", Threshold: 5000, Icon: "⬟", Label: "Email Monk", Desc: "5,000 emails cleared. Total inner peace."},
	{ID: "cleared_10000", Category: "cleared", Threshold: 10000, Icon: "⬡", Label: "Ascended", Desc: "10,000 emails cleared. You are the inbox."},

	// --- Emails sent ---
	{ID: "sent_1", Category: "sent", Threshold: 1, Icon: "↗", Label: "First Send", Desc: "Sent your first email from Termite"},
	{ID: "sent_10", Category: "sent", Threshold: 10, Icon: "↗", Label: "In Conversation", Desc: "10 emails sent"},
	{ID: "sent_100", Category: "sent", Threshold: 100, Icon: "✉", Label: "The Correspondent", Desc: "100 emails sent"},
	{ID: "sent_500", Category: "sent", Threshold: 500, Icon: "✉", Label: "Prolific", Desc: "500 emails sent"},
	{ID: "sent_1000", Category: "sent", Threshold: 1000, Icon: "✦", Label: "The Networker", Desc: "1,000 emails sent"},

	// --- Inbox zeros ---
	{ID: "zero_1", Category: "zero", Threshold: 1, Icon: "○", Label: "First Zero", Desc: "Reached inbox zero for the first time"},
	{ID: "zero_7", Category: "zero", Threshold: 7, Icon: "◎", Label: "Weekly Zero", Desc: "Inbox zero 7 times"},
	{ID: "zero_30", Category: "zero", Threshold: 30, Icon: "◉", Label: "Monthly Zero", Desc: "Inbox zero 30 times"},
	{ID: "zero_100", Category: "zero", Threshold: 100, Icon: "✦", Label: "The Minimalist", Desc: "Inbox zero 100 times. A way of life."},

	// --- Streaks ---
	{ID: "streak_3", Category: "streak", Threshold: 3, Icon: "~", Label: "3-Day Streak", Desc: "3 consecutive days reaching inbox zero"},
	{ID: "streak_7", Category: "streak", Threshold: 7, Icon: "≈", Label: "Week Streak", Desc: "7-day inbox zero streak"},
	{ID: "streak_14", Category: "streak", Threshold: 14, Icon: "≋", Label: "Fortnight", Desc: "14-day streak. This is a practice now."},
	{ID: "streak_30", Category: "streak", Threshold: 30, Icon: "∿", Label: "The Ritual", Desc: "30-day streak. You've made peace with email."},
	{ID: "streak_100", Category: "streak", Threshold: 100, Icon: "∞", Label: "Infinite Zero", Desc: "100-day streak. You may never be bothered again."},
}

// MilestonesByCategory returns all milestone definitions filtered by category.
func MilestonesByCategory(category string) []MilestoneDef {
	var result []MilestoneDef
	for _, m := range MilestoneDefinitions {
		if m.Category == category {
			result = append(result, m)
		}
	}
	return result
}
