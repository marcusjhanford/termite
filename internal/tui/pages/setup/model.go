package setuppage

import (
	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/config"
)

// Step represents the current step in the setup wizard.
type Step int

const (
	StepSelectProvider Step = iota
	StepEnterEmail
	StepAuthenticate
	StepInitialSync
	StepDone
	numSteps // sentinel
)

// authDoneMsg is sent when the background auth command finishes.
type authDoneMsg struct{ err error }

// Model is the first-run setup wizard model.
type Model struct {
	cfg *config.Config

	width  int
	height int

	step           Step
	providerChoice string // "gmail", "outlook", "fastmail", "generic"

	emailInput  string // email address being typed
	emailCursor int    // cursor position within emailInput

	authStatus string // status message shown during auth
	authDone   bool   // whether auth completed (success or failure)
	err        error
}

// New creates a default setup page model at the first step.
func New(cfg *config.Config) Model {
	return Model{
		cfg:  cfg,
		step: StepSelectProvider,
	}
}

// CanExit reports whether the user can safely leave the setup wizard right now.
// This is true on the first two steps and after auth completes (success or error).
func (m Model) CanExit() bool {
	switch m.step {
	case StepSelectProvider, StepEnterEmail:
		return true
	case StepAuthenticate:
		return m.authDone
	case StepDone:
		return true
	}
	return false
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
