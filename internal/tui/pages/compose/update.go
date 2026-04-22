package composepage

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/debuglog"
)

// Update implements tea.Model for the compose page.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.pickingFile {
			m.filePicker.SetHeight(m.filePickerHeight())
		}
		return m, nil

	case tea.KeyPressMsg:
		// --- File picker mode: route everything to the picker ---
		if m.pickingFile {
			// Esc closes the picker without selecting.
			if msg.String() == "esc" {
				m.pickingFile = false
				return m, nil
			}

			// Check for file selection before updating.
			if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
				m.attachments = append(m.attachments, path)
				m.pickingFile = false
				return m, nil
			}

			var cmd tea.Cmd
			m.filePicker, cmd = m.filePicker.Update(msg)
			return m, cmd
		}

		keyStr := msg.String()
		k := msg.Key()

		// #region agent log
		if strings.Contains(keyStr, "enter") || k.Mod != 0 {
			debuglog.AgentLog("H1-H4", "compose/update.go:KeyPressMsg", "compose key (enter/mod)", map[string]any{
				"keyStr": keyStr, "code": int(k.Code), "mod": int(k.Mod), "sendMatch": isSendComposerKey(k, keyStr),
			})
		}
		// #endregion

		// --- Normal compose mode ---

		// Cmd/Super/Meta+Enter (macOS) or Ctrl+Enter (Windows/Linux) sends the message.
		if isSendComposerKey(k, keyStr) {
			return m, m.sendCmd()
		}

		// Ctrl+A opens the in-TUI file picker for attachments.
		if keyStr == "ctrl+a" {
			m.filePicker = newFilePicker(m.filePickerHeight())
			m.pickingFile = true
			return m, m.filePicker.Init()
		}

		// Escape — let parent handle page navigation.
		if keyStr == "esc" {
			return m, nil
		}

		// Tab: accept autocomplete match, or cycle to next field.
		if keyStr == "tab" {
			if isEmailField(m.activeField) && len(m.emailMatches) > 0 {
				m.setActiveValue(m.emailMatches[0])
				m.emailMatches = nil
			} else {
				m.activeField = (m.activeField + 1) % fieldCount
				m.updateEmailMatches()
			}
			return m, nil
		}
		if keyStr == "shift+tab" {
			m.activeField = (m.activeField - 1 + fieldCount) % fieldCount
			m.updateEmailMatches()
			return m, nil
		}

		// Enter: newline in body, advance field otherwise.
		if keyStr == "enter" {
			if m.activeField == fieldBody {
				m.body += "\n"
			} else {
				m.activeField = (m.activeField + 1) % fieldCount
				m.updateEmailMatches()
			}
			return m, nil
		}

		// Alt/Ctrl/Cmd/Super + Backspace: delete last word (same as Superhuman-style clients).
		if isModifierBackspace(k) {
			m.setActiveValue(deleteWordBackward(m.activeValue()))
			m.updateEmailMatches()
			return m, nil
		}

		// Backspace.
		if keyStr == "backspace" {
			m.deleteCharFromActive()
			m.updateEmailMatches()
			return m, nil
		}

		// Regular character input — single printable character or space.
		// Bubble Tea v2 reports the space key as "space", not " ".
		if len(keyStr) == 1 || keyStr == " " || keyStr == "space" {
			ch := keyStr
			if keyStr == "space" {
				ch = " "
			}
			m.appendToActive(ch)
			m.updateEmailMatches()
			return m, nil
		}

	default:
		// Non-key messages (e.g. readDirMsg from the filepicker) need
		// to reach the picker even outside of a KeyPressMsg.
		if m.pickingFile {
			var cmd tea.Cmd
			m.filePicker, cmd = m.filePicker.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// filePickerHeight returns the height available for the file picker overlay.
func (m Model) filePickerHeight() int {
	h := m.height - 8
	if h < 5 {
		h = 5
	}
	return h
}

// sendCmd creates a tea.Cmd that emits a SendMsg.
func (m Model) sendCmd() tea.Cmd {
	return func() tea.Msg {
		return SendMsg{
			To:          m.to,
			Cc:          m.cc,
			Bcc:         m.bcc,
			Subject:     m.subject,
			Body:        m.body,
			Attachments: append([]string(nil), m.attachments...),
		}
	}
}

// appendToActive appends a character to the currently focused field.
func (m *Model) appendToActive(ch string) {
	switch m.activeField {
	case fieldTo:
		m.to += ch
	case fieldCc:
		m.cc += ch
	case fieldBcc:
		m.bcc += ch
	case fieldSubject:
		m.subject += ch
	case fieldBody:
		m.body += ch
	}
}

// setActiveValue replaces the value of the active field.
func (m *Model) setActiveValue(val string) {
	switch m.activeField {
	case fieldTo:
		m.to = val
	case fieldCc:
		m.cc = val
	case fieldBcc:
		m.bcc = val
	case fieldSubject:
		m.subject = val
	case fieldBody:
		m.body = val
	}
}

// activeValue returns the current value of the active field.
func (m *Model) activeValue() string {
	switch m.activeField {
	case fieldTo:
		return m.to
	case fieldCc:
		return m.cc
	case fieldBcc:
		return m.bcc
	case fieldSubject:
		return m.subject
	case fieldBody:
		return m.body
	}
	return ""
}

// deleteCharFromActive removes the last rune from the focused field.
func (m *Model) deleteCharFromActive() {
	switch m.activeField {
	case fieldTo:
		m.to = dropLastRune(m.to)
	case fieldCc:
		m.cc = dropLastRune(m.cc)
	case fieldBcc:
		m.bcc = dropLastRune(m.bcc)
	case fieldSubject:
		m.subject = dropLastRune(m.subject)
	case fieldBody:
		m.body = dropLastRune(m.body)
	}
}

// dropLastRune removes the last rune from a string, handling unicode correctly.
func dropLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

// updateEmailMatches filters knownEmails by the current field value.
// Only active when the active field is an email field.
func (m *Model) updateEmailMatches() {
	m.emailMatches = nil
	if !isEmailField(m.activeField) || len(m.knownEmails) == 0 {
		return
	}
	input := strings.ToLower(m.activeValue())
	if input == "" {
		return
	}
	for _, email := range m.knownEmails {
		if strings.HasPrefix(strings.ToLower(email), input) && strings.ToLower(email) != input {
			m.emailMatches = append(m.emailMatches, email)
		}
	}
}
