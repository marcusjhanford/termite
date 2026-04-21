package notifications

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

// DesktopNotifier sends native desktop notifications using the beeep library.
// It supports macOS (via osascript), Linux (via notify-send/libnotify),
// and Windows (via toast notifications).
type DesktopNotifier struct{}

// Notify sends a desktop notification with the given title and body.
// The notification uses the system's native notification mechanism.
func (d *DesktopNotifier) Notify(title, body string) error {
	// beeep.Notify sends a native desktop notification.
	// The third argument is the icon path — empty string uses a default.
	if err := beeep.Notify(title, body, ""); err != nil {
		return fmt.Errorf("desktop notify: %w", err)
	}
	return nil
}
