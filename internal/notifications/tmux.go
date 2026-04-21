package notifications

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TmuxNotifier manages tmux window title updates to show unread counts.
type TmuxNotifier struct{}

// InTmux reports whether the current process is running inside a tmux session.
func (t *TmuxNotifier) InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// SetTitle updates the tmux window title to include the unread count.
// If unreadCount is 0, the title is set to just "termite".
// If not running in tmux, this is a no-op.
func (t *TmuxNotifier) SetTitle(unreadCount int) {
	if !t.InTmux() {
		return
	}

	var title string
	if unreadCount > 0 {
		title = fmt.Sprintf("termite (%d)", unreadCount)
	} else {
		title = "termite"
	}

	// Use tmux rename-window to set the current window title.
	cmd := exec.Command("tmux", "rename-window", title)
	cmd.Run() //nolint:errcheck // best effort
}

// ResetTitle restores the tmux window title to automatic naming.
// Called on application exit.
func (t *TmuxNotifier) ResetTitle() {
	if !t.InTmux() {
		return
	}

	// Re-enable automatic window renaming.
	cmd := exec.Command("tmux", "set-window-option", "automatic-rename", "on")
	cmd.Run() //nolint:errcheck // best effort
}

// CurrentPane returns the current tmux pane ID, or empty string if not in tmux.
func (t *TmuxNotifier) CurrentPane() string {
	if !t.InTmux() {
		return ""
	}

	cmd := exec.Command("tmux", "display-message", "-p", "#{pane_id}")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
