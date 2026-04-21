package inboxzeropage

// View renders the forest scene animation for the inbox zero celebration.
func (m Model) View() string {
	if m.scene == nil {
		return "\n\n    Inbox Zero!\n\n    Waiting for terminal dimensions..."
	}
	return m.scene.View()
}
