package commands

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Command represents a command that can be executed from the command bar.
type Command struct {
	Name        string
	Description string
	Handler     func(args string) tea.Cmd
}

// Registry manages all registered slash-commands and provides dispatch
// and completion functionality.
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry. If a command with the same name
// already exists, it is overwritten.
func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name] = cmd
}

// Dispatch parses the input string as "name args" and invokes the matching
// command's handler. Returns nil if the command is not found.
func (r *Registry) Dispatch(input string) tea.Cmd {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Strip leading colon or slash if present.
	if strings.HasPrefix(input, ":") || strings.HasPrefix(input, "/") {
		input = input[1:]
	}

	name, args, _ := strings.Cut(input, " ")
	name = strings.TrimSpace(name)
	args = strings.TrimSpace(args)

	cmd, ok := r.commands[name]
	if !ok {
		return func() tea.Msg {
			return CommandErrorMsg{Command: name, Err: "unknown command: :" + name}
		}
	}

	return cmd.Handler(args)
}

// Completions returns all commands whose names start with the given prefix,
// sorted alphabetically. An empty prefix returns all commands.
func (r *Registry) Completions(prefix string) []Command {
	prefix = strings.TrimPrefix(prefix, ":")
	prefix = strings.TrimPrefix(prefix, "/")
	prefix = strings.ToLower(prefix)

	var matches []Command
	for _, cmd := range r.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name), prefix) {
			matches = append(matches, cmd)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})
	return matches
}

// All returns every registered command sorted alphabetically by name.
func (r *Registry) All() []Command {
	cmds := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})
	return cmds
}

// CommandErrorMsg is sent when a command dispatch fails.
type CommandErrorMsg struct {
	Command string
	Err     string
}
