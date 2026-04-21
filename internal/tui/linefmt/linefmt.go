package linefmt

import (
	"encoding/json"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// CollapseWhitespace replaces newlines and runs of space with a single space.
func CollapseWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}

// FormatJSONStringList parses a JSON string array (as stored in the DB) into
// a human-readable comma-separated list. If parsing fails, returns CollapseWhitespace(s).
func FormatJSONStringList(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return CollapseWhitespace(s)
	}
	var addrs []string
	if err := json.Unmarshal([]byte(s), &addrs); err != nil || len(addrs) == 0 {
		return CollapseWhitespace(s)
	}
	return strings.Join(addrs, ", ")
}

// TruncateDisplayWidth shortens s so lipgloss.Width(s) <= maxW (including ellipsis when trimmed).
func TruncateDisplayWidth(s string, maxW int) string {
	if maxW < 2 {
		return "…"
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > maxW-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

// WrapPlainText wraps text to at most maxW terminal cells per line, breaking on spaces.
func WrapPlainText(s string, maxW int) string {
	if maxW < 8 {
		maxW = 8
	}
	var out strings.Builder
	for pi, para := range strings.Split(s, "\n") {
		if pi > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(wrapParagraph(CollapseWhitespace(para), maxW))
	}
	return out.String()
}

func wrapParagraph(para string, maxW int) string {
	words := strings.Fields(para)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
		}
	}
	for _, word := range words {
		cand := word
		if cur.Len() > 0 {
			cand = cur.String() + " " + word
		}
		if lipgloss.Width(cand) <= maxW {
			if cur.Len() > 0 {
				cur.WriteString(" ")
			}
			cur.WriteString(word)
			continue
		}
		flush()
		chunks := splitWordByWidth(word, maxW)
		for i := 0; i < len(chunks)-1; i++ {
			lines = append(lines, chunks[i])
		}
		if len(chunks) > 0 {
			cur.WriteString(chunks[len(chunks)-1])
		}
	}
	flush()
	return strings.Join(lines, "\n")
}

func splitWordByWidth(word string, maxW int) []string {
	if lipgloss.Width(word) <= maxW {
		return []string{word}
	}
	var chunks []string
	var b strings.Builder
	for _, r := range word {
		trial := b.String() + string(r)
		if lipgloss.Width(trial) > maxW {
			if b.Len() > 0 {
				chunks = append(chunks, b.String())
				b.Reset()
			}
			b.WriteRune(r)
			if lipgloss.Width(b.String()) > maxW {
				chunks = append(chunks, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		chunks = append(chunks, b.String())
	}
	return chunks
}
