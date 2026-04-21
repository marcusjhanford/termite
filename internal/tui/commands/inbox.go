package commands

import (
	tea "charm.land/bubbletea/v2"
)

// SwitchInboxMsg requests switching the active split inbox.
type SwitchInboxMsg struct {
	InboxName string
}

// InboxCommand returns the /inbox command, which switches the active
// split inbox. Usage: /inbox [name]
func InboxCommand() Command {
	return Command{
		Name:        "inbox",
		Description: "Switch to a split inbox by name",
		Handler: func(args string) tea.Cmd {
			if args == "" {
				return func() tea.Msg {
					return CommandErrorMsg{
						Command: "inbox",
						Err:     "usage: /inbox <name>",
					}
				}
			}
			name := args
			return func() tea.Msg {
				return SwitchInboxMsg{InboxName: name}
			}
		},
	}
}
