package metricsdashboard

import (
	tea "charm.land/bubbletea/v2"
)

// Update handles messages for the metrics dashboard page.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % numTabs
			return m, nil
		case "shift+tab", "h", "left":
			m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
			return m, nil
		}
	}

	return m, nil
}
