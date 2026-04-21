package milestonetoast

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Animation phases for the toast lifecycle.
type toastPhase int

const (
	PhaseHidden toastPhase = iota
	PhaseSlideIn
	PhaseHolding
	PhaseSlideOut
)

const (
	slideInDuration  = 300 * time.Millisecond
	holdDuration     = 3 * time.Second
	slideOutDuration = 300 * time.Millisecond
	tickInterval     = 50 * time.Millisecond
)

// slideInTicks is the number of ticks for the slide-in animation.
var slideInTicks = int(slideInDuration / tickInterval)

// holdTicks is the number of ticks for the hold phase.
var holdTicks = int(holdDuration / tickInterval)

// slideOutTicks is the number of ticks for the slide-out animation.
var slideOutTicks = int(slideOutDuration / tickInterval)

// MilestoneDisplay holds the data to render a milestone toast.
type MilestoneDisplay struct {
	Icon  string
	Label string
	Desc  string
}

// tickMsg is the internal animation tick message.
type tickMsg struct{}

// Model is the milestone toast overlay component.
type Model struct {
	queue    []MilestoneDisplay
	current  *MilestoneDisplay
	phase    toastPhase
	holdTick int
	width    int
	height   int
}

// New creates a hidden milestone toast model.
func New() Model {
	return Model{
		phase: PhaseHidden,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles animation ticks for the toast.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		switch m.phase {
		case PhaseSlideIn:
			m.holdTick++
			if m.holdTick >= slideInTicks {
				m.phase = PhaseHolding
				m.holdTick = 0
			}
			return m, m.tick()

		case PhaseHolding:
			m.holdTick++
			if m.holdTick >= holdTicks {
				m.phase = PhaseSlideOut
				m.holdTick = 0
			}
			return m, m.tick()

		case PhaseSlideOut:
			m.holdTick++
			if m.holdTick >= slideOutTicks {
				m.current = nil
				m.phase = PhaseHidden
				m.holdTick = 0
				// Process next in queue.
				return m, m.dequeue()
			}
			return m, m.tick()
		}
	}

	return m, nil
}

// View renders the toast overlay. Returns empty string when hidden.
func (m Model) View() string {
	if m.phase == PhaseHidden || m.current == nil {
		return ""
	}

	iconStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ABABAB"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2).
		Background(lipgloss.Color("#1A1A2E"))

	// Build toast content.
	line1 := iconStyle.Render(m.current.Icon) + " " + labelStyle.Render(m.current.Label)
	line2 := descStyle.Render(m.current.Desc)
	content := lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	toast := boxStyle.Render(content)

	// Calculate horizontal offset based on phase for slide animation.
	toastWidth := lipgloss.Width(toast)
	var xOffset int

	switch m.phase {
	case PhaseSlideIn:
		// Slide from right edge to final position.
		progress := float64(m.holdTick) / float64(slideInTicks)
		offScreen := toastWidth + 2
		xOffset = int(float64(offScreen) * (1.0 - progress))
	case PhaseHolding:
		xOffset = 0
	case PhaseSlideOut:
		// Slide out to the right.
		progress := float64(m.holdTick) / float64(slideOutTicks)
		offScreen := toastWidth + 2
		xOffset = int(float64(offScreen) * progress)
	}

	// Position at bottom-right.
	rightMargin := 2 + xOffset
	bottomMargin := 2

	// Pad the toast to position it at bottom-right.
	leftPad := m.width - toastWidth - rightMargin
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := m.height - 4 - bottomMargin // 4 = approximate toast height
	if topPad < 0 {
		topPad = 0
	}

	positioned := strings.Repeat("\n", topPad) +
		strings.Repeat(" ", leftPad) + toast

	return positioned
}

// Queue adds a milestone to the display queue. If nothing is currently
// showing, it starts the animation immediately.
func (m *Model) Queue(milestone MilestoneDisplay) {
	m.queue = append(m.queue, milestone)
}

// StartIfIdle starts showing the next queued toast if the toast is
// currently hidden. Call this after Queue to begin the animation.
func (m *Model) StartIfIdle() tea.Cmd {
	if m.phase == PhaseHidden && m.current == nil {
		return m.dequeue()
	}
	return nil
}

// SetSize sets the available width and height for positioning.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Visible returns whether the toast is currently visible.
func (m Model) Visible() bool {
	return m.phase != PhaseHidden
}

// dequeue starts showing the next milestone from the queue.
func (m *Model) dequeue() tea.Cmd {
	if len(m.queue) == 0 {
		m.current = nil
		m.phase = PhaseHidden
		return nil
	}

	next := m.queue[0]
	m.queue = m.queue[1:]
	m.current = &next
	m.phase = PhaseSlideIn
	m.holdTick = 0

	return m.tick()
}

// tick returns a command that sends a tickMsg after the tick interval.
func (m Model) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
