package inboxzeropage

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/animation"
	"github.com/termite-mail/termite/internal/tui/msgs"
)

// Update handles messages for the inbox zero celebration page.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// (Re-)create the forest scene at the new dimensions.
		m.scene = animation.NewForestScene(m.width, m.height, m.theme)
		return m, nil

	case tickMsg:
		if m.scene == nil {
			// No scene yet — wait for a resize.
			return m, tick(m.tickInterval)
		}

		m.scene.Tick()

		// Adjust tick interval based on phase.
		switch m.scene.Phase() {
		case animation.PhaseGrowing:
			m.tickInterval = 30 * time.Millisecond
		case animation.PhaseSettling:
			m.tickInterval = 60 * time.Millisecond
		case animation.PhaseAlive:
			m.tickInterval = 80 * time.Millisecond
		case animation.PhaseFading:
			if m.scene.FadeDone() {
				// Fade complete — navigate back to the main page.
				return m, func() tea.Msg {
					return msgs.NavigateMsg{Page: "main"}
				}
			}
			m.tickInterval = 40 * time.Millisecond
		}

		return m, tick(m.tickInterval)

	case tea.KeyPressMsg:
		if m.scene == nil {
			return m, nil
		}
		// Any key press during the alive phase begins the fade.
		phase := m.scene.Phase()
		if phase == animation.PhaseAlive || phase == animation.PhaseSettling {
			m.scene.BeginFade()
		}
		// During fading, keys are ignored (let the fade finish).
		// During growing, keys are ignored (let the tree finish).
		return m, nil
	}

	return m, nil
}
