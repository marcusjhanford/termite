package inboxzeropage

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/animation"
	"github.com/termite-mail/termite/internal/themes"
)

// tickMsg drives the animation loop.
type tickMsg struct{}

// Model is the inbox zero celebration page. It wraps a ForestScene
// from the animation package and drives it with a tick loop.
type Model struct {
	width  int
	height int

	scene *animation.ForestScene
	theme *themes.Theme

	// tickInterval controls the animation speed.
	// Faster during growth, slower once alive.
	tickInterval time.Duration
}

// New creates a new inbox zero page model. The scene is not created until
// Init or the first WindowSizeMsg, since we need terminal dimensions.
func New(theme *themes.Theme) Model {
	return Model{
		theme:        theme,
		tickInterval: 30 * time.Millisecond,
	}
}

// Init starts the animation tick loop.
func (m Model) Init() tea.Cmd {
	return tick(m.tickInterval)
}

// tick returns a tea.Cmd that sends a tickMsg after the given duration.
func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
