package messageview

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/termite-mail/termite/internal/tui/linefmt"
)

// Model is the right-pane message content viewer.
type Model struct {
	headers string
	body    string
	scrollY int
	focused bool
	width   int
	height  int
}

// New creates an empty message view model.
func New() Model {
	return Model{}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles scrolling input for the message view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.scrollDown()
		case "k", "up":
			m.scrollUp()
		case "g":
			// Jump to top.
			m.scrollY = 0
		case "G":
			// Jump to bottom.
			m.scrollY = m.maxScroll()
		}
	}

	return m, nil
}

// View renders the message headers and scrollable body.
func (m Model) View() string {
	if m.headers == "" && m.body == "" {
		return m.emptyView()
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	headerLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ABABAB")).
		Bold(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DDDDDD"))

	var rows []string

	// Render headers (truncate values so one header never spans the whole terminal width).
	headerLines := strings.Split(m.headers, "\n")
	for _, line := range headerLines {
		if parts := strings.SplitN(line, ": ", 2); len(parts) == 2 {
			val := linefmt.TruncateDisplayWidth(parts[1], m.width-12)
			label := headerLabelStyle.Render(parts[0] + ":")
			value := headerStyle.Render(" " + val)
			rows = append(rows, label+value)
		} else {
			rows = append(rows, headerLabelStyle.Render(linefmt.TruncateDisplayWidth(line, m.width-2)))
		}
	}

	// Separator.
	sepWidth := m.width
	if sepWidth > 0 {
		rows = append(rows, separatorStyle.Render(strings.Repeat("─", sepWidth)))
	}

	// Calculate how many body lines we can display.
	headerHeight := len(rows)
	bodyHeight := m.height - headerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Split body into lines (wrapped to pane width) and apply scroll.
	bodyLines := m.wrappedBodyLines()

	// Clamp scroll.
	if m.scrollY > len(bodyLines)-bodyHeight {
		scrollMax := len(bodyLines) - bodyHeight
		if scrollMax < 0 {
			scrollMax = 0
		}
		m.scrollY = scrollMax
	}

	end := m.scrollY + bodyHeight
	if end > len(bodyLines) {
		end = len(bodyLines)
	}

	visibleBody := bodyLines[m.scrollY:end]
	for _, line := range visibleBody {
		rows = append(rows, bodyStyle.Render(line))
	}

	// Scroll indicator.
	if len(bodyLines) > bodyHeight {
		pos := fmt.Sprintf(" [%d/%d] ", m.scrollY+1, len(bodyLines))
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render(pos)
		rows = append(rows, scrollIndicator)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// SetMessage sets the message to display.
func (m *Model) SetMessage(from, to, subject, date, body string) {
	var headerParts []string
	headerParts = append(headerParts, "From: "+from)
	headerParts = append(headerParts, "To: "+linefmt.FormatJSONStringList(to))
	headerParts = append(headerParts, "Subject: "+subject)
	headerParts = append(headerParts, "Date: "+date)
	m.headers = strings.Join(headerParts, "\n")
	m.body = body
	m.scrollY = 0
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

func (m *Model) scrollDown() {
	max := m.maxScroll()
	if m.scrollY < max {
		m.scrollY++
	}
}

func (m *Model) scrollUp() {
	if m.scrollY > 0 {
		m.scrollY--
	}
}

func (m Model) bodyWrapWidth() int {
	w := m.width - 2
	if w < 8 {
		return 8
	}
	return w
}

func (m Model) wrappedBodyLines() []string {
	if m.body == "" {
		return []string{""}
	}
	return strings.Split(linefmt.WrapPlainText(m.body, m.bodyWrapWidth()), "\n")
}

func (m Model) maxScroll() int {
	bodyLines := m.wrappedBodyLines()
	// Reserve space for headers (4 lines) + separator (1 line).
	bodyHeight := m.height - 5
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	max := len(bodyLines) - bodyHeight
	if max < 0 {
		return 0
	}
	return max
}

func (m Model) emptyView() string {
	placeholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render("Select a thread to view")
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(placeholder)
}
