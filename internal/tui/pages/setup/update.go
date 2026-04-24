package setuppage

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/providers"
	"github.com/termite-mail/termite/internal/tui/msgs"
)

// runAuthCmd returns a tea.Cmd that performs the provider auth flow in the
// background. On completion it sends an authDoneMsg back to the update loop.
func runAuthCmd(provider string, email string, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		p, err := providers.NewProvider(provider)
		if err != nil {
			return authDoneMsg{err: err}
		}

		creds, err := p.RunAuthFlow(email)
		if err != nil {
			return authDoneMsg{err: err}
		}

		accountEmail := creds.IdentityEmail
		if accountEmail == "" {
			accountEmail = creds.Username
		}
		if accountEmail == "" {
			accountEmail = email
		}

		// Persist account to config on success (OAuth may replace typed hint with provider email).
		acct := config.AccountConfig{
			ID:       accountEmail,
			Name:     provider,
			Email:    accountEmail,
			Provider: provider,
		}
		cfg.Accounts = append(cfg.Accounts, acct)
		if err := config.Save(cfg, ""); err != nil {
			return authDoneMsg{err: fmt.Errorf("failed to save config: %w", err)}
		}

		return authDoneMsg{err: nil}
	}
}

// Update implements tea.Model for the setup wizard.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case authDoneMsg:
		m.authDone = true
		if msg.err != nil {
			m.err = msg.err
			m.authStatus = "Authentication failed."
		} else {
			m.err = nil
			m.authStatus = "Authenticated!"
			m.step = StepInitialSync
		}
		return m, nil

	case tea.KeyPressMsg:
		keyStr := msg.String()

		switch m.step {
		case StepSelectProvider:
			return m.updateSelectProvider(keyStr)
		case StepEnterEmail:
			return m.updateEnterEmail(keyStr)
		case StepAuthenticate:
			return m.updateAuthenticate(keyStr)
		case StepInitialSync:
			if keyStr == "enter" {
				m.step = StepDone
			}
			return m, nil
		case StepDone:
			if keyStr == "enter" {
				return m, func() tea.Msg {
					return msgs.NavigateMsg{Page: "main", KickInitialSync: true}
				}
			}
		}
	}

	return m, nil
}

// updateSelectProvider handles input on the provider selection step.
// Only Gmail is supported on launch; other providers show as "coming soon".
func (m Model) updateSelectProvider(keyStr string) (Model, tea.Cmd) {
	switch keyStr {
	case "1":
		m.providerChoice = "gmail"
		m.step = StepEnterEmail
	}
	return m, nil
}

// updateEnterEmail handles text input for the email address.
func (m Model) updateEnterEmail(keyStr string) (Model, tea.Cmd) {
	switch keyStr {
	case "enter":
		if m.emailInput == "" {
			return m, nil
		}
		// Advance to auth step and kick off the background auth flow.
		m.step = StepAuthenticate
		m.authStatus = "Opening your browser to sign in…"
		m.authDone = false
		m.err = nil
		return m, runAuthCmd(m.providerChoice, m.emailInput, m.cfg)

	case "backspace":
		if len(m.emailInput) > 0 {
			runes := []rune(m.emailInput)
			m.emailInput = string(runes[:len(runes)-1])
		}
		return m, nil

	case "esc":
		// Go back to provider selection.
		m.step = StepSelectProvider
		m.emailInput = ""
		m.emailCursor = 0
		return m, nil

	default:
		// Append printable characters.
		if len(keyStr) == 1 || keyStr == " " {
			m.emailInput += keyStr
		}
		return m, nil
	}
}

// updateAuthenticate handles input while waiting for auth or after an error.
func (m Model) updateAuthenticate(keyStr string) (Model, tea.Cmd) {
	if !m.authDone {
		// Auth is still running — ignore keys.
		return m, nil
	}

	// Auth finished with an error: let user retry or go back.
	if m.err != nil {
		switch keyStr {
		case "enter":
			// Retry auth.
			m.authDone = false
			m.err = nil
			m.authStatus = "Opening your browser to sign in…"
			return m, runAuthCmd(m.providerChoice, m.emailInput, m.cfg)
		case "esc":
			// Go back to email entry.
			m.step = StepEnterEmail
			m.authDone = false
			m.err = nil
			m.authStatus = ""
			return m, nil
		}
	}

	return m, nil
}
