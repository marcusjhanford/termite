package inboxlist

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// InboxItem represents a single inbox/label entry.
type InboxItem struct {
	ID          string
	Label       string
	UnreadCount int
}

// InboxSelectedMsg is emitted when the user selects a different inbox.
type InboxSelectedMsg struct {
	InboxID string
}

// Model is the left-pane inbox list component.
type Model struct {
	inboxes       []InboxItem
	selected      int
	activeInboxID string // the inbox whose threads are currently displayed
	focused       bool
	width         int
	height        int
}

// New creates an inbox list model with the given items.
func New(inboxes []InboxItem) Model {
	return Model{
		inboxes:  inboxes,
		selected: 0,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input for the inbox list.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.inboxes)-1 {
				m.selected++
				return m, m.emitSelected()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				return m, m.emitSelected()
			}
		}
	}

	return m, nil
}

// View renders the inbox list.
func (m Model) View() string {
	if len(m.inboxes) == 0 {
		return m.emptyView()
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ABABAB"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	if m.focused {
		selectedStyle = selectedStyle.
			Background(lipgloss.Color("#7D56F4"))
	}

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6F61")).
		Bold(true)

	countZeroStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	var rows []string
	rows = append(rows, titleStyle.Render("Inboxes"))

	for i, inbox := range m.inboxes {
		isActive := inbox.ID == m.activeInboxID
		isSelected := i == m.selected

		// Bullet: filled circle if unread, empty circle otherwise.
		bullet := "○"
		if inbox.UnreadCount > 0 {
			bullet = "●"
		}

		prefix := bullet
		if isActive {
			prefix = "▸"
		}

		label := fmt.Sprintf(" %s %s", prefix, inbox.Label)

		// Count badge — always shown, bright when unread, dim when zero.
		badgeStyle := countZeroStyle
		if inbox.UnreadCount > 0 {
			badgeStyle = countStyle
		}
		badge := badgeStyle.Render(fmt.Sprintf(" (%d)", inbox.UnreadCount))

		style := normalStyle
		if isSelected {
			style = selectedStyle
		} else if isActive {
			style = activeStyle
		}

		// Render the label with available width, then append badge.
		row := style.Render(label) + badge

		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Constrain to allocated size.
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// SetFocused sets whether this component is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the available width and height for rendering.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetActiveInbox sets the inbox ID that is currently being viewed.
// It is visually distinguished from the selected (cursor) inbox.
func (m *Model) SetActiveInbox(id string) {
	m.activeInboxID = id
}

// SelectedInbox returns the ID of the currently selected inbox.
func (m Model) SelectedInbox() string {
	if len(m.inboxes) == 0 {
		return ""
	}
	return m.inboxes[m.selected].ID
}

func (m Model) emptyView() string {
	placeholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render("No inboxes")
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(placeholder)
}

func (m Model) emitSelected() tea.Cmd {
	id := m.inboxes[m.selected].ID
	return func() tea.Msg {
		return InboxSelectedMsg{InboxID: id}
	}
}

// clamp keeps v within [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
