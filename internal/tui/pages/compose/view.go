package composepage

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// View implements tea.Model. It renders a full-screen compose form.
// When the file picker is active it replaces the compose form with the picker.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading compose..."
	}

	// If the file picker is open, render it inside a bordered box instead
	// of the compose form.
	if m.pickingFile {
		return m.filePickerView()
	}

	if m.embedded {
		return m.viewEmbedded()
	}

	formWidth := clamp(m.width-4, 40, 80)
	inputWidth := formWidth - 16 // account for label width + padding + borders

	// Styles.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Width(10).
		Foreground(lipgloss.Color("#ABABAB")).
		Align(lipgloss.Right).
		PaddingRight(1)

	activeLabelStyle := lipgloss.NewStyle().
		Width(10).
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Align(lipgloss.Right).
		PaddingRight(1)

	inputBg := lipgloss.Color("#1e1e2e")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(inputBg).
		Width(inputWidth)

	activeInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(inputBg).
		Width(inputWidth)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Background(inputBg)

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	cursor := "\u2588" // █

	// Title based on mode.
	title := modeTitle(m.mode)
	header := titleStyle.Render(title)

	// Single-line fields.
	fields := []struct {
		label string
		value string
		idx   int
	}{
		{"To:", m.to, fieldTo},
		{"Cc:", m.cc, fieldCc},
		{"Bcc:", m.bcc, fieldBcc},
		{"Subject:", m.subject, fieldSubject},
	}

	var rows []string
	rows = append(rows, header)

	for _, f := range fields {
		isActive := m.activeField == f.idx && !m.pickingFile
		lbl := labelStyle
		inp := inputStyle
		if isActive {
			lbl = activeLabelStyle
			inp = activeInputStyle
		}

		label := lbl.Render(f.label)

		val := f.value
		if isActive {
			// Show cursor and possible autocomplete hint.
			if isEmailField(f.idx) && len(m.emailMatches) > 0 && strings.HasPrefix(strings.ToLower(m.emailMatches[0]), strings.ToLower(f.value)) {
				remaining := m.emailMatches[0][len(f.value):]
				val = val + cursor + hintStyle.Render(remaining)
				rawInp := activeInputStyle.Copy().UnsetWidth()
				rendered := rawInp.Render(val)
				row := lipgloss.JoinHorizontal(lipgloss.Top, label, rendered)
				rows = append(rows, row)
				continue
			}
			val += cursor
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, label, inp.Render(val))
		rows = append(rows, row)
	}

	// Separator before body.
	rows = append(rows, separatorStyle.Render(strings.Repeat("─", clamp(formWidth-4, 10, 60))))

	// Body field (multi-line).
	{
		isActive := m.activeField == fieldBody && !m.pickingFile
		lbl := labelStyle
		inp := inputStyle
		if isActive {
			lbl = activeLabelStyle
			inp = activeInputStyle
		}

		label := lbl.Render("Body:")

		bodyVal := m.body
		if isActive {
			bodyVal += cursor
		}

		bodyLines := strings.Split(bodyVal, "\n")
		maxBodyLines := clamp(m.height-14-len(m.attachments), 3, 20)
		if len(bodyLines) > maxBodyLines {
			bodyLines = bodyLines[len(bodyLines)-maxBodyLines:]
		}

		colSpacer := lipgloss.NewStyle().Width(10).Render("")
		for i, line := range bodyLines {
			if i == 0 {
				rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, inp.Render(line)))
				continue
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, colSpacer, inp.Render(line)))
		}
	}

	// Attachments list.
	if len(m.attachments) > 0 {
		rows = append(rows, "")
		attLabelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ABABAB")).
			Bold(true)
		rows = append(rows, attLabelStyle.Render("Attachments:"))

		attStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#58a6ff"))
		for _, path := range m.attachments {
			rows = append(rows, attStyle.Render("  📎 "+path))
		}
	}

	// Help footer.
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		MarginTop(1)
	rows = append(rows, helpStyle.Render("Tab next field | Enter newline (body) | Ctrl+A attach | Ctrl+Enter or Ctrl+S send (⌘+Enter often blocked by Terminal) | Esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Width(formWidth).
		Padding(1, 2).
		Margin(2, 0)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}

// viewEmbedded is a dense top-aligned layout for the message-column reply area.
func (m Model) viewEmbedded() string {
	labelW := 8
	inputW := m.width - labelW - 2
	if inputW < 8 {
		inputW = 8
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	labelStyle := lipgloss.NewStyle().
		Width(labelW).
		Foreground(lipgloss.Color("#ABABAB")).
		Align(lipgloss.Right).
		PaddingRight(1)

	activeLabelStyle := lipgloss.NewStyle().
		Width(labelW).
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Align(lipgloss.Right).
		PaddingRight(1)

	inputBg := lipgloss.Color("#1e1e2e")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cccccc")).
		Background(inputBg).
		Width(inputW)

	activeInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(inputBg).
		Width(inputW)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Background(inputBg)

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	cursor := "\u2588"

	title := titleStyle.Render(modeTitle(m.mode))

	fields := []struct {
		label string
		value string
		idx   int
	}{
		{"To:", m.to, fieldTo},
		{"Cc:", m.cc, fieldCc},
		{"Bcc:", m.bcc, fieldBcc},
		{"Subject:", m.subject, fieldSubject},
	}

	var rows []string
	rows = append(rows, title)

	for _, f := range fields {
		isActive := m.activeField == f.idx
		lbl := labelStyle
		inp := inputStyle
		if isActive {
			lbl = activeLabelStyle
			inp = activeInputStyle
		}
		label := lbl.Render(f.label)
		val := f.value
		if isActive {
			if isEmailField(f.idx) && len(m.emailMatches) > 0 && strings.HasPrefix(strings.ToLower(m.emailMatches[0]), strings.ToLower(f.value)) {
				remaining := m.emailMatches[0][len(f.value):]
				val = val + cursor + hintStyle.Render(remaining)
				rawInp := activeInputStyle.Copy().UnsetWidth()
				rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, rawInp.Render(val)))
				continue
			}
			val += cursor
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, inp.Render(val)))
	}

	rows = append(rows, separatorStyle.Render(strings.Repeat("─", clamp(m.width-2, 6, inputW+labelW))))

	// Body
	{
		isActive := m.activeField == fieldBody
		lbl := labelStyle
		inp := inputStyle
		if isActive {
			lbl = activeLabelStyle
			inp = activeInputStyle
		}
		label := lbl.Render("Body:")
		bodyVal := m.body
		if isActive {
			bodyVal += cursor
		}
		bodyLines := strings.Split(bodyVal, "\n")
		maxBodyLines := m.height - 10
		if maxBodyLines < 2 {
			maxBodyLines = 2
		}
		if len(bodyLines) > maxBodyLines {
			bodyLines = bodyLines[len(bodyLines)-maxBodyLines:]
		}
		colSpacer := lipgloss.NewStyle().Width(labelW).Render("")
		for i, line := range bodyLines {
			if i == 0 {
				rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, inp.Render(line)))
				continue
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, colSpacer, inp.Render(line)))
		}
	}

	if len(m.attachments) > 0 {
		rows = append(rows, "")
		attStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))
		for _, path := range m.attachments {
			rows = append(rows, attStyle.Render("  "+path))
		}
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	rows = append(rows, helpStyle.Render("Tab • Enter • Ctrl+Enter / Ctrl+S send • Esc"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Width(m.width).
		MaxHeight(m.height).
		Padding(0, 1).
		Render(content)
}

// filePickerView renders the in-TUI file browser overlay.
func (m Model) filePickerView() string {
	formWidth := clamp(m.width-4, 40, 80)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		MarginTop(1)

	var rows []string
	rows = append(rows, titleStyle.Render("Attach a file"))
	rows = append(rows, pathStyle.Render(m.filePicker.CurrentDirectory))
	rows = append(rows, "")
	rows = append(rows, m.filePicker.View())
	rows = append(rows, helpStyle.Render("j/k navigate | Enter select | h/Backspace back | Esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Width(formWidth).
		Padding(1, 2).
		Margin(2, 0)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}

// clamp restricts v to the range [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// modeTitle returns a display string for the compose mode.
func modeTitle(mode string) string {
	switch mode {
	case "reply":
		return "Reply"
	case "replyall":
		return "Reply All"
	case "forward":
		return "Forward"
	default:
		return "New Message"
	}
}

// fieldName returns a display name for the given field index (for debugging).
func fieldName(idx int) string {
	switch idx {
	case fieldTo:
		return "To"
	case fieldCc:
		return "Cc"
	case fieldBcc:
		return "Bcc"
	case fieldSubject:
		return "Subject"
	case fieldBody:
		return "Body"
	default:
		return fmt.Sprintf("Field(%d)", idx)
	}
}
