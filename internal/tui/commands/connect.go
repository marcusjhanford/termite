package commands

import (
	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/tui/msgs"
)

// ConnectCommand returns the /connect command, which navigates to the
// account setup page.
func ConnectCommand() Command {
	return Command{
		Name:        "connect",
		Description: "Add or reconnect an email account",
		Handler: func(args string) tea.Cmd {
			return func() tea.Msg {
				return msgs.NavigateMsg{Page: "setup"}
			}
		},
	}
}
