package composepage

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/debuglog"
)

// isSendComposerKey is true for Ctrl+Enter (Windows/Linux), Cmd+Enter / Super+Enter (macOS), or Meta+Enter.
func isSendComposerKey(k tea.Key, keyStr string) bool {
	// #region agent log
	ok := false
	switch keyStr {
	case "ctrl+enter", "super+enter", "meta+enter", "ctrl+s":
		// ctrl+s: reliable send shortcut on macOS where Cmd+Enter is often captured by the terminal.
		ok = true
	default:
		ok = k.Code == tea.KeyEnter && (k.Mod&(tea.ModCtrl|tea.ModSuper|tea.ModMeta)) != 0
	}
	if k.Code == tea.KeyEnter || k.Mod != 0 || strings.Contains(keyStr, "enter") {
		debuglog.AgentLog("H2-H3", "compose/edit.go:isSendComposerKey", "send key evaluation", map[string]any{
			"keyStr": keyStr, "code": int(k.Code), "mod": int(k.Mod), "match": ok,
		})
	}
	// #endregion
	return ok
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
