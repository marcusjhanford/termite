package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// SwitchInboxMsg requests switching the active split inbox.
type SwitchInboxMsg struct {
	InboxName string
}

// CreateInboxMsg requests creating a new split inbox.
type CreateInboxMsg struct {
	ID    string
	Label string
}

// DeleteInboxMsg requests deleting a split inbox.
type DeleteInboxMsg struct {
	ID string
}

// ListInboxesMsg requests listing split inboxes.
type ListInboxesMsg struct{}

// InboxCommand returns the /inbox command.
// Usage:
//   /inbox <name>           — switch to inbox
//   /inbox create <name>    — create a new split inbox
//   /inbox delete <name>    — delete a split inbox
//   /inbox list             — list split inboxes
func InboxCommand() Command {
	return Command{
		Name:        "inbox",
		Description: "Manage split inboxes",
		Handler: func(args string) tea.Cmd {
			parts := strings.Fields(strings.TrimSpace(args))
			if len(parts) == 0 {
				return func() tea.Msg {
					return CommandErrorMsg{
						Command: "inbox",
						Err:     "usage: /inbox [create|delete|list] <name>",
					}
				}
			}

			sub := strings.ToLower(parts[0])

			switch sub {
			case "create":
				if len(parts) < 2 {
					return func() tea.Msg {
						return CommandErrorMsg{
							Command: "inbox",
							Err:     "usage: /inbox create <name>",
						}
					}
				}
				name := strings.Join(parts[1:], " ")
				return func() tea.Msg {
					return CreateInboxMsg{ID: slugify(name), Label: name}
				}

			case "delete":
				if len(parts) < 2 {
					return func() tea.Msg {
						return CommandErrorMsg{
							Command: "inbox",
							Err:     "usage: /inbox delete <name>",
						}
					}
				}
				return func() tea.Msg {
					return DeleteInboxMsg{ID: slugify(parts[1])}
				}

			case "list":
				return func() tea.Msg {
					return ListInboxesMsg{}
				}

			default:
				// Switch to inbox by name.
				return func() tea.Msg {
					return SwitchInboxMsg{InboxName: sub}
				}
			}
		},
	}
}

// slugify creates a simple ID slug from a display name.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	id := b.String()
	if id == "" {
		id = "inbox"
	}
	return id
}
