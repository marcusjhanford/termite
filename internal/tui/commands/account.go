package commands

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/tui/msgs"
)

// SwitchAccountMsg requests switching the active account.
type SwitchAccountMsg struct {
	AccountID string
}

// ListAccountsMsg requests listing all accounts.
type ListAccountsMsg struct{}

// OpenAccountPickerMsg requests opening the account picker overlay.
type OpenAccountPickerMsg struct{}

// AccountCommand returns the /account command.
// Usage:
//   /account                — open the account picker overlay
//   /account list           — list all configured accounts
//   /account switch <id>    — switch to the given account
//   /account add            — go to setup to add a new account
func AccountCommand() Command {
	return Command{
		Name:        "account",
		Description: "Manage accounts",
		Handler: func(args string) tea.Cmd {
			parts := strings.Fields(strings.TrimSpace(args))
			if len(parts) == 0 {
				return func() tea.Msg {
					return OpenAccountPickerMsg{}
				}
			}

			sub := strings.ToLower(parts[0])

			switch sub {
			case "list":
				return func() tea.Msg {
					return ListAccountsMsg{}
				}
			case "switch":
				if len(parts) < 2 {
					return func() tea.Msg {
						return CommandErrorMsg{
							Command: "account",
							Err:     "usage: /account switch <id>",
						}
					}
				}
				return func() tea.Msg {
					return SwitchAccountMsg{AccountID: parts[1]}
				}
			case "add":
				return func() tea.Msg {
					return msgs.NavigateMsg{Page: "setup"}
				}
			default:
				return func() tea.Msg {
					return CommandErrorMsg{
						Command: "account",
						Err:     fmt.Sprintf("unknown subcommand: %s", sub),
					}
				}
			}
		},
	}
}
