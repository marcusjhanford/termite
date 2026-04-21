package tui

import (
	"charm.land/bubbles/v2/key"

	"github.com/termite-mail/termite/internal/config"
)

// KeyMap defines all keybindings for the Termite TUI.
type KeyMap struct {
	Compose    key.Binding
	Reply      key.Binding
	ReplyAll   key.Binding
	Forward    key.Binding
	Archive    key.Binding
	Delete     key.Binding
	MarkRead   key.Binding
	MarkUnread key.Binding
	Snooze     key.Binding
	Next       key.Binding
	Prev       key.Binding
	Open       key.Binding
	Zero       key.Binding
	Search     key.Binding
	Command    key.Binding
	Quit       key.Binding
	Tab        key.Binding
	ShiftTab   key.Binding
	Escape     key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Compose: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compose"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		ReplyAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reply all"),
		),
		Forward: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "forward"),
		),
		Archive: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "archive"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		MarkRead: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark read"),
		),
		MarkUnread: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "mark unread"),
		),
		Snooze: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "snooze"),
		),
		Next: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "previous"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Zero: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "inbox zero"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// BuildKeyMap constructs a KeyMap from the user's config, falling back
// to defaults for any unset binding.
func BuildKeyMap(cfg *config.Config) KeyMap {
	km := DefaultKeyMap()
	kb := cfg.Keybindings

	if kb.Compose != "" {
		km.Compose.SetKeys(kb.Compose)
	}
	if kb.Reply != "" {
		km.Reply.SetKeys(kb.Reply)
	}
	if kb.ReplyAll != "" {
		km.ReplyAll.SetKeys(kb.ReplyAll)
	}
	if kb.Forward != "" {
		km.Forward.SetKeys(kb.Forward)
	}
	if kb.Archive != "" {
		km.Archive.SetKeys(kb.Archive)
	}
	if kb.Delete != "" {
		km.Delete.SetKeys(kb.Delete)
	}
	if kb.MarkRead != "" {
		km.MarkRead.SetKeys(kb.MarkRead)
	}
	if kb.MarkUnread != "" {
		km.MarkUnread.SetKeys(kb.MarkUnread)
	}
	if kb.Snooze != "" {
		km.Snooze.SetKeys(kb.Snooze)
	}
	if kb.Next != "" {
		km.Next.SetKeys(kb.Next)
	}
	if kb.Prev != "" {
		km.Prev.SetKeys(kb.Prev)
	}
	if kb.Open != "" {
		km.Open.SetKeys(kb.Open)
	}
	if kb.Zero != "" {
		km.Zero.SetKeys(kb.Zero)
	}
	if kb.Search != "" {
		km.Search.SetKeys(kb.Search)
	}
	if kb.Command != "" {
		km.Command.SetKeys(kb.Command)
	}
	if kb.Quit != "" {
		km.Quit.SetKeys(kb.Quit)
	}

	return km
}
