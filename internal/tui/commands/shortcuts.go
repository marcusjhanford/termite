package commands

import (
	tea "charm.land/bubbletea/v2"
)

// ShortcutsMsg requests displaying the keybinding cheatsheet.
type ShortcutsMsg struct{}

// ShortcutsCommand returns the /shortcuts command, which displays the
// current keybinding cheatsheet overlay.
func ShortcutsCommand() Command {
	return Command{
		Name:        "shortcuts",
		Description: "Show keybinding cheatsheet",
		Handler: func(args string) tea.Cmd {
			return func() tea.Msg {
				return ShortcutsMsg{}
			}
		},
	}
}
