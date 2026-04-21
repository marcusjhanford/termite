package composepage

import (
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// isSendComposerKey is true for Ctrl+Enter (Windows/Linux), Cmd+Enter / Super+Enter (macOS), or Meta+Enter.
func isSendComposerKey(k tea.Key, keyStr string) bool {
	switch keyStr {
	case "ctrl+enter", "super+enter", "meta+enter":
		return true
	}
	return k.Code == tea.KeyEnter && (k.Mod&(tea.ModCtrl|tea.ModSuper|tea.ModMeta)) != 0
}

// isModifierBackspace is Backspace combined with Ctrl, Alt, Super/Meta (Cmd on macOS), etc.
func isModifierBackspace(k tea.Key) bool {
	if k.Code != tea.KeyBackspace {
		return false
	}
	return k.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModSuper|tea.ModMeta) != 0
}

// deleteWordBackward removes the last whitespace-delimited word, matching typical mail clients
// (Option/Alt+Backspace, Ctrl+Backspace, Cmd+Backspace).
func deleteWordBackward(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	i := len(r) - 1
	for i >= 0 && unicode.IsSpace(r[i]) {
		i--
	}
	if i < 0 {
		return ""
	}
	for i >= 0 && !unicode.IsSpace(r[i]) {
		i--
	}
	return string(r[:i+1])
}
