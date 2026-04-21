package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// DaemonStatusMsg requests checking the daemon's current status.
type DaemonStatusMsg struct{}

// DaemonStartMsg requests starting the background sync daemon.
type DaemonStartMsg struct{}

// DaemonStopMsg requests stopping the background sync daemon.
type DaemonStopMsg struct{}

// DaemonCommand returns the /daemon command.
// Usage: /daemon         — show daemon status
// Usage: /daemon start   — start the background daemon
// Usage: /daemon stop    — stop the background daemon
func DaemonCommand() Command {
	return Command{
		Name:        "daemon",
		Description: "Control the background sync daemon",
		Handler: func(args string) tea.Cmd {
			args = strings.TrimSpace(strings.ToLower(args))

			switch args {
			case "start":
				return func() tea.Msg {
					return DaemonStartMsg{}
				}
			case "stop":
				return func() tea.Msg {
					return DaemonStopMsg{}
				}
			case "", "status":
				return func() tea.Msg {
					return DaemonStatusMsg{}
				}
			default:
				return func() tea.Msg {
					return CommandErrorMsg{
						Command: "daemon",
						Err:     "usage: /daemon [start|stop|status]",
					}
				}
			}
		},
	}
}
