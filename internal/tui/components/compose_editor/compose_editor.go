package composeeditor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// SendMsg is emitted when the user submits the compose form.
type SendMsg struct {
	To      string
	Subject string
	Body    string
}

// Field represents a single form field in the compose editor.
type Field struct {
	Label  string
	Value  string
	IsBody bool // body field is multi-line
}

// Model is the compose form editor component.
type Model struct {
	fields    []Field
	activeIdx int
	focused   bool
	width     int
	height    int
	mode      string // "new", "reply", "replyall", "forward"
}

// New creates a compose editor for the given mode.
func New(mode string) Model {
	fields := buildFields(mode)

	// Default focus to body for reply modes, To for others.
	activeIdx := 0
	switch mode {
	case "reply", "replyall":
		// Focus body since To/Subject are pre-filled.
		for i, f := range fields {
			if f.IsBody {
				activeIdx = i
				break
			}
		}
	}

	return Model{
		fields:    fields,
		activeIdx: activeIdx,
		mode:      mode,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input for the compose editor.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		keyStr := msg.String()

		switch keyStr {
		case "tab":
			m.activeIdx = (m.activeIdx + 1) % len(m.fields)
			return m, nil
		case "shift+tab":
			m.activeIdx = (m.activeIdx - 1 + len(m.fields)) % len(m.fields)
			return m, nil

		case "ctrl+enter":
			// Submit the form.
			return m, func() tea.Msg {
				return SendMsg{
					To:      m.To(),
					Subject: m.Subject(),
					Body:    m.Body(),
				}
			}

		case "backspace":
			m.deleteChar()
			return m, nil

		case "enter":
			// In body field, insert newline. Otherwise move to next field.
			if m.fields[m.activeIdx].IsBody {
				m.fields[m.activeIdx].Value += "\n"
			} else {
				m.activeIdx = (m.activeIdx + 1) % len(m.fields)
			}
			return m, nil

		default:
			// Append printable characters.
			if len(keyStr) == 1 || keyStr == " " {
				m.fields[m.activeIdx].Value += keyStr
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the compose form.
func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Width(10).
		Foreground(lipgloss.Color("#ABABAB")).
		Align(lipgloss.Right).
		PaddingRight(1)

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	activeInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333"))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	cursor := "█"

	title := modeTitle(m.mode)
	var rows []string
	rows = append(rows, titleStyle.Render(title))

	for i, field := range m.fields {
		if field.IsBody {
			// Separator before body.
			sepWidth := m.width - 4
			if sepWidth < 10 {
				sepWidth = 10
			}
			if sepWidth > 60 {
				sepWidth = 60
			}
			rows = append(rows, separatorStyle.Render(strings.Repeat("─", sepWidth)))
		}

		label := labelStyle.Render(field.Label + ":")
		val := field.Value
		style := inputStyle

		if m.focused && i == m.activeIdx {
			val += cursor
			style = activeInputStyle
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, label, style.Render(val))
		rows = append(rows, row)
	}

	// Help line.
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		MarginTop(1)
	rows = append(rows, helpStyle.Render("Tab: next field | Shift+Tab: prev | Ctrl+Enter: send | Esc: cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// SetSize sets the available width and height.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused sets whether this component is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// To returns the To field value.
func (m Model) To() string {
	for _, f := range m.fields {
		if f.Label == "To" {
			return f.Value
		}
	}
	return ""
}

// Subject returns the Subject field value.
func (m Model) Subject() string {
	for _, f := range m.fields {
		if f.Label == "Subject" {
			return f.Value
		}
	}
	return ""
}

// Body returns the Body field value.
func (m Model) Body() string {
	for _, f := range m.fields {
		if f.IsBody {
			return f.Value
		}
	}
	return ""
}

// deleteChar removes the last character from the active field.
func (m *Model) deleteChar() {
	val := m.fields[m.activeIdx].Value
	if len(val) == 0 {
		return
	}
	runes := []rune(val)
	m.fields[m.activeIdx].Value = string(runes[:len(runes)-1])
}

// buildFields creates the field list for a given compose mode.
func buildFields(mode string) []Field {
	fields := []Field{
		{Label: "To", Value: ""},
		{Label: "Cc", Value: ""},
		{Label: "Bcc", Value: ""},
	}

	switch mode {
	case "reply":
		fields = append(fields, Field{Label: "Subject", Value: "Re: "})
	case "replyall":
		fields = append(fields, Field{Label: "Subject", Value: "Re: "})
	case "forward":
		fields = append(fields, Field{Label: "Subject", Value: "Fwd: "})
	default:
		fields = append(fields, Field{Label: "Subject", Value: ""})
	}

	fields = append(fields, Field{Label: "Body", Value: "", IsBody: true})
	return fields
}

// modeTitle returns the display title for the compose mode.
func modeTitle(mode string) string {
	switch mode {
	case "reply":
		return "Reply"
	case "replyall":
		return "Reply All"
	case "forward":
		return "Forward"
	default:
		return "New Message"
	}
}
