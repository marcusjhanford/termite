package themes

import (
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
)

// Styles holds pre-built lipgloss styles for every UI component in Termite.
// Rebuild via BuildStyles whenever the active theme changes.
type Styles struct {
	// Thread list
	ThreadUnread   lipgloss.Style
	ThreadRead     lipgloss.Style
	ThreadSelected lipgloss.Style
	ThreadPreview  lipgloss.Style
	ThreadDate     lipgloss.Style

	// Unread indicator (the dot / marker)
	UnreadDot lipgloss.Style

	// Message viewer
	MessageHeader  lipgloss.Style
	MessageBody    lipgloss.Style
	MessageQuote   lipgloss.Style
	MessageLink    lipgloss.Style
	MessageDivider lipgloss.Style

	// Inbox sidebar
	InboxLabel       lipgloss.Style
	InboxLabelActive lipgloss.Style
	InboxBadge       lipgloss.Style

	// Command bar
	CommandBarInput lipgloss.Style
	CommandBarMatch lipgloss.Style
	CommandBarHint  lipgloss.Style

	// Status bar
	StatusBar      lipgloss.Style
	StatusBarKey   lipgloss.Style
	StatusBarValue lipgloss.Style

	// Borders
	Border        lipgloss.Style
	BorderFocused lipgloss.Style

	// Milestone / toast
	MilestoneToast lipgloss.Style

	// Metrics bar
	MetricsBar      lipgloss.Style
	MetricsLabel    lipgloss.Style
	MetricsValue    lipgloss.Style
	MetricsPositive lipgloss.Style

	// Semantic feedback
	Success lipgloss.Style
	Warning lipgloss.Style
	Danger  lipgloss.Style
	Info    lipgloss.Style

	// General purpose
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Muted    lipgloss.Style
}

// c is a shorthand that converts a hex color string to a color.Color via lipgloss.
func c(hex string) color.Color {
	if hex == "" {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(hex)
}

// BuildStyles constructs a complete Styles set from a Theme.
func BuildStyles(t *Theme) Styles {
	var s Styles

	// ── Thread list ───────────────────────────────────────────────────

	s.ThreadUnread = lipgloss.NewStyle().
		Foreground(c(t.UnreadSubject)).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	s.ThreadRead = lipgloss.NewStyle().
		Foreground(c(t.ReadSubject)).
		PaddingLeft(1).
		PaddingRight(1)

	s.ThreadSelected = lipgloss.NewStyle().
		Foreground(c(t.SelectionText)).
		Background(c(t.Selection)).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	s.ThreadPreview = lipgloss.NewStyle().
		Foreground(c(t.ReadPreview)).
		Italic(true)

	s.ThreadDate = lipgloss.NewStyle().
		Foreground(c(t.TextMuted))

	s.UnreadDot = lipgloss.NewStyle().
		Foreground(c(t.UnreadIndicator)).
		Bold(true)

	// ── Message viewer ────────────────────────────────────────────────

	s.MessageHeader = lipgloss.NewStyle().
		Foreground(c(t.Text)).
		Bold(true).
		PaddingBottom(1).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(c(t.Border))

	s.MessageBody = lipgloss.NewStyle().
		Foreground(c(t.Text)).
		PaddingLeft(2).
		PaddingRight(2)

	s.MessageQuote = lipgloss.NewStyle().
		Foreground(c(t.TextMuted)).
		Italic(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(c(t.Border)).
		PaddingLeft(1)

	s.MessageLink = lipgloss.NewStyle().
		Foreground(c(t.Primary)).
		Underline(true)

	s.MessageDivider = lipgloss.NewStyle().
		Foreground(c(t.Border))

	// ── Inbox sidebar ─────────────────────────────────────────────────

	s.InboxLabel = lipgloss.NewStyle().
		Foreground(c(t.TextMuted)).
		PaddingLeft(2).
		PaddingRight(2)

	s.InboxLabelActive = lipgloss.NewStyle().
		Foreground(c(t.Primary)).
		Bold(true).
		PaddingLeft(2).
		PaddingRight(2)

	s.InboxBadge = lipgloss.NewStyle().
		Foreground(c(t.Background)).
		Background(c(t.Primary)).
		Bold(true).
		Padding(0, 1)

	// ── Command bar ───────────────────────────────────────────────────

	s.CommandBarInput = lipgloss.NewStyle().
		Foreground(c(t.Text)).
		Background(c(t.CommandBarBg)).
		PaddingLeft(1).
		PaddingRight(1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(c(t.CommandBorder))

	s.CommandBarMatch = lipgloss.NewStyle().
		Foreground(c(t.Accent)).
		Bold(true)

	s.CommandBarHint = lipgloss.NewStyle().
		Foreground(c(t.TextDim)).
		Italic(true)

	// ── Status bar ────────────────────────────────────────────────────

	s.StatusBar = lipgloss.NewStyle().
		Foreground(c(t.StatusBarText)).
		Background(c(t.StatusBarBg)).
		Padding(0, 1)

	s.StatusBarKey = lipgloss.NewStyle().
		Foreground(c(t.Primary)).
		Background(c(t.StatusBarBg)).
		Bold(true).
		PaddingRight(1)

	s.StatusBarValue = lipgloss.NewStyle().
		Foreground(c(t.StatusBarText)).
		Background(c(t.StatusBarBg))

	// ── Borders ───────────────────────────────────────────────────────

	s.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(t.Border))

	s.BorderFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(t.BorderFocus))

	// ── Milestone toast ───────────────────────────────────────────────

	s.MilestoneToast = lipgloss.NewStyle().
		Foreground(c(t.Background)).
		Background(c(t.Success)).
		Bold(true).
		Padding(0, 2).
		Margin(1)

	// ── Metrics bar ───────────────────────────────────────────────────

	s.MetricsBar = lipgloss.NewStyle().
		Foreground(c(t.TextMuted)).
		Background(c(t.SurfaceAlt)).
		Padding(0, 1)

	s.MetricsLabel = lipgloss.NewStyle().
		Foreground(c(t.TextMuted)).
		PaddingRight(1)

	s.MetricsValue = lipgloss.NewStyle().
		Foreground(c(t.Text)).
		Bold(true)

	s.MetricsPositive = lipgloss.NewStyle().
		Foreground(c(t.Success)).
		Bold(true)

	// ── Semantic ──────────────────────────────────────────────────────

	s.Success = lipgloss.NewStyle().Foreground(c(t.Success))
	s.Warning = lipgloss.NewStyle().Foreground(c(t.Warning))
	s.Danger = lipgloss.NewStyle().Foreground(c(t.Danger))
	s.Info = lipgloss.NewStyle().Foreground(c(t.Info))

	// ── General purpose ───────────────────────────────────────────────

	s.Title = lipgloss.NewStyle().
		Foreground(c(t.Text)).
		Bold(true).
		PaddingBottom(1)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(c(t.TextMuted)).
		Italic(true)

	s.Muted = lipgloss.NewStyle().
		Foreground(c(t.TextDim))

	return s
}
