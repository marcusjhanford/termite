package composepage

import (
	"os"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
)

// Field index constants for Tab navigation.
const (
	fieldTo = iota
	fieldCc
	fieldBcc
	fieldSubject
	fieldBody
	fieldCount // sentinel — total number of fields
)

// Model is the compose page model for drafting emails.
type Model struct {
	width  int
	height int

	mode     string // "new", "reply", "replyall", "forward"
	threadID string // set for reply/forward modes

	// Field values.
	to      string
	cc      string
	bcc     string
	subject string
	body    string

	// Attachments.
	attachments []string // file paths
	pickingFile bool     // true while the in-TUI file picker is open
	filePicker  filepicker.Model

	// Email autocomplete.
	knownEmails  []string // populated from DB
	emailMatches []string // current filtered matches

	activeField int

	// embedded: compact layout when reply/forward is shown in the message column.
	embedded bool
}

// SetEmbedded toggles the compact inline layout (vs full-screen centered compose).
func (m *Model) SetEmbedded(v bool) {
	m.embedded = v
}

// SendMsg is emitted when the user presses Cmd/Super+Enter (macOS) or Ctrl+Enter (Windows/Linux).
type SendMsg struct {
	To          string
	Cc          string
	Bcc         string
	Subject     string
	Body        string
	Attachments []string
}

// New creates a blank compose model.
func New() Model {
	return Model{
		mode:        "new",
		activeField: fieldTo,
	}
}

// NewWithMode creates a compose model pre-configured for the given mode.
func NewWithMode(mode, threadID string) Model {
	m := New()
	m.mode = mode
	m.threadID = threadID

	switch mode {
	case "reply", "replyall":
		m.subject = "Re: "
		m.activeField = fieldBody
	case "forward":
		m.subject = "Fwd: "
		m.activeField = fieldTo
	}

	return m
}

// SetKnownEmails populates the autocomplete list from external data.
func (m *Model) SetKnownEmails(emails []string) {
	m.knownEmails = make([]string, len(emails))
	copy(m.knownEmails, emails)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// newFilePicker creates a filepicker.Model rooted at the user's home directory.
func newFilePicker(height int) filepicker.Model {
	fp := filepicker.New()
	fp.AutoHeight = false
	fp.SetHeight(height)
	fp.FileAllowed = true
	fp.DirAllowed = false
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}
	return fp
}

// isEmailField returns true if the given field index is To, Cc, or Bcc.
func isEmailField(idx int) bool {
	return idx == fieldTo || idx == fieldCc || idx == fieldBcc
}
