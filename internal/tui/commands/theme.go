package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ThemeChangeMsg requests applying a new theme by ID.
type ThemeChangeMsg struct {
	ThemeID string
}

// ThemeListMsg requests listing all available themes.
type ThemeListMsg struct{}

// ThemeCommand returns the /theme command.
// Usage: /theme list   — list available themes
// Usage: /theme [name] — switch to the named theme
func ThemeCommand() Command {
	return Command{
		Name:        "theme",
		Description: "Switch theme or list available themes",
		Handler: func(args string) tea.Cmd {
			args = strings.TrimSpace(args)
			if args == "" || args == "list" {
				return func() tea.Msg {
					return ThemeListMsg{}
				}
			}
			id := args
			return func() tea.Msg {
				return ThemeChangeMsg{ThemeID: id}
			}
		},
	}
}
