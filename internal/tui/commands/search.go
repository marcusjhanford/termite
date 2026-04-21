package commands

import (
	tea "charm.land/bubbletea/v2"
)

// SearchMsg requests a full-text search across all messages.
type SearchMsg struct {
	Query string
}

// SearchCommand returns the /search command. Usage: /search [query]
func SearchCommand() Command {
	return Command{
		Name:        "search",
		Description: "Search messages by keyword (FTS5)",
		Handler: func(args string) tea.Cmd {
			if args == "" {
				return func() tea.Msg {
					return CommandErrorMsg{
						Command: "search",
						Err:     "usage: /search <query>",
					}
				}
			}
			query := args
			return func() tea.Msg {
				return SearchMsg{Query: query}
			}
		},
	}
}
