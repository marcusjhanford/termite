package commandbar

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// CommandMsg is emitted when the user submits a command.
type CommandMsg struct {
	Command string
}

// SearchMsg is emitted when the user submits a search query.
type SearchMsg struct {
	Query string
}

// CancelledMsg is emitted when the user cancels the command bar.
type CancelledMsg struct{}

// Mode distinguishes between command and search input.
type Mode int

const (
	ModeCommand Mode = iota
	ModeSearch
)

// Model is the bottom command/search input bar.
type Model struct {
	input    string
	active   bool
	mode     Mode
	width    int
	matches  []string
	commands []string // registered command names for autocomplete
}

// New creates an inactive command bar model.
func New() Model {
	return Model{}
}

// SetCommands sets the list of available command names for autocomplete.
// Call this after building the command registry.
func (m *Model) SetCommands(names []string) {
	m.commands = make([]string, len(names))
	copy(m.commands, names)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input for the command bar when active.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		keyStr := msg.String()

		switch keyStr {
		case "esc":
			m.Deactivate()
			return m, func() tea.Msg { return CancelledMsg{} }

		case "enter":
			input := strings.TrimSpace(m.input)
			mode := m.mode
			m.Deactivate()

			if input == "" {
				return m, nil
			}

			if mode == ModeSearch {
				return m, func() tea.Msg { return SearchMsg{Query: input} }
			}
			return m, func() tea.Msg { return CommandMsg{Command: input} }

		case "backspace":
			if len(m.input) > 0 {
				runes := []rune(m.input)
				m.input = string(runes[:len(runes)-1])
			}
			m.updateMatches()
			return m, nil

		case "tab":
			// Autocomplete: apply first match.
			if len(m.matches) > 0 {
				m.input = m.matches[0]
				m.updateMatches()
			}
			return m, nil

		default:
			// Append printable characters.
			if len(keyStr) == 1 || keyStr == " " {
				m.input += keyStr
				m.updateMatches()
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the command bar.
func (m Model) View() string {
	if !m.active {
		return ""
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	matchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	cursor := "█"
	var prefix string
	var label string

	switch m.mode {
	case ModeSearch:
		prefix = "/"
		label = " search"
	case ModeCommand:
		prefix = ":"
		label = " command"
	}

	prompt := promptStyle.Render(prefix)
	text := inputStyle.Render(m.input + cursor)
	line := prompt + text

	// Inline autocomplete hint (command mode only).
	if m.mode == ModeCommand && len(m.matches) > 0 {
		if strings.HasPrefix(m.matches[0], m.input) {
			remaining := strings.TrimPrefix(m.matches[0], m.input)
			if remaining != "" {
				line += matchStyle.Render(remaining)
			}
		} else {
			line += matchStyle.Render("  " + m.matches[0])
		}
	}

	// Right-aligned mode label.
	lineWidth := lipgloss.Width(line)
	labelRendered := labelStyle.Render(label)
	labelWidth := lipgloss.Width(labelRendered)
	gap := m.width - lineWidth - labelWidth
	if gap < 1 {
		gap = 1
	}

	return line + strings.Repeat(" ", gap) + labelRendered
}

// Activate enables the command bar in command mode.
func (m *Model) Activate() {
	m.active = true
	m.mode = ModeCommand
	m.input = ""
	m.matches = nil
}

// ActivateSearch enables the command bar in search mode.
func (m *Model) ActivateSearch() {
	m.active = true
	m.mode = ModeSearch
	m.input = ""
	m.matches = nil
}

// Deactivate hides the command bar and clears state.
func (m *Model) Deactivate() {
	m.active = false
	m.input = ""
	m.mode = ModeCommand
	m.matches = nil
}

// IsActive returns whether the command bar is currently active.
func (m Model) IsActive() bool {
	return m.active
}

// SetWidth sets the rendering width.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// updateMatches filters registered commands by current input prefix.
// Only active in command mode.
func (m *Model) updateMatches() {
	if m.mode != ModeCommand || m.input == "" {
		m.matches = nil
		return
	}

	lower := strings.ToLower(m.input)
	var matches []string
	for _, cmd := range m.commands {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			matches = append(matches, cmd)
		}
	}
	m.matches = matches
}
