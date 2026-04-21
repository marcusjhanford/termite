package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/tui/msgs"
)

// MetricsExportMsg requests exporting metrics to a file.
type MetricsExportMsg struct {
	Format string // "json" or "csv"
}

// MetricsCommand returns the /metrics command.
// Usage: /metrics            — navigate to the metrics dashboard
// Usage: /metrics export json — export metrics as JSON
// Usage: /metrics export csv  — export metrics as CSV
func MetricsCommand() Command {
	return Command{
		Name:        "metrics",
		Description: "View metrics dashboard or export data",
		Handler: func(args string) tea.Cmd {
			args = strings.TrimSpace(args)

			// No arguments: navigate to the metrics dashboard page.
			if args == "" {
				return func() tea.Msg {
					return msgs.NavigateMsg{Page: "metrics"}
				}
			}

			parts := strings.Fields(args)

			// /metrics export [json|csv]
			if parts[0] == "export" {
				format := "json"
				if len(parts) > 1 {
					format = strings.ToLower(parts[1])
				}
				if format != "json" && format != "csv" {
					return func() tea.Msg {
						return CommandErrorMsg{
							Command: "metrics",
							Err:     "unsupported export format: " + format + " (use json or csv)",
						}
					}
				}
				return func() tea.Msg {
					return MetricsExportMsg{Format: format}
				}
			}

			return func() tea.Msg {
				return CommandErrorMsg{
					Command: "metrics",
					Err:     "usage: /metrics [export json|csv]",
				}
			}
		},
	}
}
