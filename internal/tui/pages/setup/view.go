package setuppage

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
)

// View implements tea.Model. It renders the setup wizard step-by-step.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading setup..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true)

	var content string

	switch m.step {
	case StepSelectProvider:
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Welcome to Termite"),
			bodyStyle.Render("Select your email provider:"),
			"",
			bodyStyle.Render("  [1] Gmail (bring your own OAuth)"),
			mutedStyle.Render("  [2] Outlook (coming soon)"),
			mutedStyle.Render("  [3] Fastmail (coming soon)"),
			mutedStyle.Render("  [4] Generic IMAP (coming soon)"),
			"",
			mutedStyle.Render("Press 1 to continue with Gmail, Esc to go back."),
		)

	case StepEnterEmail:
		cursor := "█"
		emailLine := bodyStyle.Render("  Email: ") + bodyStyle.Render(m.emailInput+cursor)

		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Enter your email address"),
			"",
			emailLine,
			"",
			mutedStyle.Render("Press Enter to continue, Esc to go back."),
		)

	case StepAuthenticate:
		if m.authDone && m.err != nil {
			// Auth failed — show error with retry options.
			content = lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render("Authentication Failed"),
				"",
				errorStyle.Render("Error: "+m.err.Error()),
				"",
				mutedStyle.Render("Press Enter to retry, or Esc to go back."),
			)
		} else {
			// Auth in progress.
			provider := m.providerChoice
			if provider == "" {
				provider = "your provider"
			}
			spinner := "◐"

			content = lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render(fmt.Sprintf("Authenticating with %s...", provider)),
				"",
				bodyStyle.Render("Your browser will open so you can sign in with your provider."),
				bodyStyle.Render("Approve access there, then return here when the browser says you are done."),
				"",
				mutedStyle.Render("If you built from source, make sure you have set"),
				mutedStyle.Render("TERMITE_GMAIL_CLIENT_ID and TERMITE_GMAIL_CLIENT_SECRET."),
				"",
				bodyStyle.Render(spinner+" "+m.authStatus),
			)
		}

	case StepInitialSync:
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Initial Sync"),
			"",
			bodyStyle.Render("Initial sync will happen after setup."),
			"",
			mutedStyle.Render("Press Enter to continue."),
		)

	case StepDone:
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Setup Complete"),
			bodyStyle.Render("Your account has been configured."),
			"",
			bodyStyle.Render("Press Enter to start using Termite."),
		)
	}

	// Step indicator.
	stepNum := int(m.step) + 1
	stepText := mutedStyle.Render(fmt.Sprintf("Step %d of %d", stepNum, int(numSteps)))

	formWidth := min(m.width-4, 60)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Width(formWidth).
		Padding(1, 2).
		Margin(1, 0)

	inner := lipgloss.JoinVertical(lipgloss.Left, content, "", stepText)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(inner),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
