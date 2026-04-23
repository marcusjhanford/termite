package composepage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/termite-mail/termite/internal/db"
)

// FromThreadDraft builds a compose model with recipients and subject from the latest
// message in a thread. The body is left empty so the user types above the visible message.
// latest may be nil (caller keeps defaults).
// defaultAccountID pre-selects the account to send from.
// accountSignature is the user's configured signature for the selected account.
func FromThreadDraft(mode, threadID string, latest *db.Message, accountEmails []string, defaultAccountID string, accountSignature string) Model {
	m := NewWithMode(mode, threadID)

	m.SetAccountEmails(accountEmails, defaultAccountID)

	self := normalizeSet(accountEmails)

	switch mode {
	case "reply":
		m.to = strings.TrimSpace(latest.FromAddr)
		m.cc = ""
		m.subject = normalizeReplySubject(latest.Subject)
		m.body = buildComposeBody(mode, latest, accountSignature)
		m.activeField = fieldBody
	case "replyall":
		to, cc := replyAllRecipients(latest, self)
		m.to = to
		m.cc = cc
		m.subject = normalizeReplySubject(latest.Subject)
		m.body = buildComposeBody(mode, latest, accountSignature)
		m.activeField = fieldBody
	case "forward":
		m.to = ""
		m.cc = ""
		m.subject = normalizeForwardSubject(latest.Subject)
		m.body = buildComposeBody(mode, latest, accountSignature)
		m.activeField = fieldTo
	case "new":
		m.body = buildComposeBody(mode, latest, accountSignature)
	}

	return m
}

func normalizeSet(emails []string) map[string]bool {
	out := make(map[string]bool)
	for _, e := range emails {
		n := normalizeEmail(e)
		if n != "" {
			out[n] = true
		}
	}
	return out
}

func normalizeEmail(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "<"); i >= 0 {
		if j := strings.LastIndex(s, ">"); j > i {
			s = s[i+1 : j]
		}
	}
	return strings.ToLower(strings.TrimSpace(s))
}

func parseAddressList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var addrs []string
		if err := json.Unmarshal([]byte(s), &addrs); err == nil {
			return addrs
		}
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func uniqueAddrs(addrs []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		n := normalizeEmail(a)
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, a)
	}
	return out
}

func replyAllRecipients(msg *db.Message, self map[string]bool) (to string, cc string) {
	from := strings.TrimSpace(msg.FromAddr)
	fromNorm := normalizeEmail(from)

	var pool []string
	pool = append(pool, parseAddressList(msg.ToAddrs)...)
	pool = append(pool, parseAddressList(msg.CcAddrs)...)
	pool = uniqueAddrs(append(pool, from))

	var participants []string
	for _, a := range pool {
		if self[normalizeEmail(a)] {
			continue
		}
		participants = append(participants, a)
	}

	if !self[fromNorm] && from != "" {
		// Typical case: reply to author; everyone else on Cc.
		to = from
		var rest []string
		for _, a := range participants {
			if normalizeEmail(a) == fromNorm {
				continue
			}
			rest = append(rest, a)
		}
		return to, strings.Join(rest, ", ")
	}

	// Sender is self: address remaining recipients.
	if len(participants) == 0 {
		return "", ""
	}
	to = participants[0]
	if len(participants) > 1 {
		cc = strings.Join(participants[1:], ", ")
	}
	return to, cc
}

// buildComposeBody assembles the pre-filled body text for a compose draft.
// For new messages: just the signature block.
// For replies/forwards: signature block + quoted original message history.
func buildComposeBody(mode string, latest *db.Message, accountSignature string) string {
	var parts []string

	// Leading blank line so the user has space to type before the signature.
	parts = append(parts, "")

	// Signature block (user signature + termite signature).
	sigBlock := buildSignatureBlock(accountSignature)
	if sigBlock != "" {
		parts = append(parts, sigBlock)
	}

	// Quoted history for replies and forwards.
	if latest != nil && mode != "new" {
		history := formatQuotedHistory(mode, latest)
		if history != "" {
			parts = append(parts, history)
		}
	}

	return strings.Join(parts, "\n")
}

// buildSignatureBlock creates the signature footer.
// If an account signature is configured it appears first; the termite
// signature is always appended.
func buildSignatureBlock(accountSignature string) string {
	termiteSig := "Sent with termite (https://github.com/marcusjhanford/termite)"

	var parts []string
	parts = append(parts, "--")

	if strings.TrimSpace(accountSignature) != "" {
		parts = append(parts, strings.TrimSpace(accountSignature))
	}
	parts = append(parts, termiteSig)

	return strings.Join(parts, "\n")
}

// formatQuotedHistory formats the original message as quoted text.
func formatQuotedHistory(mode string, msg *db.Message) string {
	var parts []string

	dateStr := formatDateForQuote(msg.Date)
	sender := strings.TrimSpace(msg.FromAddr)

	switch mode {
	case "reply", "replyall":
		parts = append(parts, fmt.Sprintf("On %s, %s wrote:", dateStr, sender))
	case "forward":
		parts = append(parts, "---------- Forwarded message ----------")
		parts = append(parts, fmt.Sprintf("From: %s", sender))
		parts = append(parts, fmt.Sprintf("Date: %s", dateStr))
		parts = append(parts, fmt.Sprintf("Subject: %s", msg.Subject))
		parts = append(parts, "")
	}

	body := strings.TrimSpace(msg.BodyText)
	if body == "" {
		body = strings.TrimSpace(msg.BodyHTML)
		// Strip simple HTML tags for plain-text quoting.
		body = stripSimpleHTML(body)
	}

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		parts = append(parts, "> "+line)
	}

	return strings.Join(parts, "\n")
}

// formatDateForQuote formats a unix timestamp into an email-style date string.
func formatDateForQuote(ts int64) string {
	t := time.Unix(ts, 0)
	return t.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}

// stripSimpleHTML removes common HTML tags for plain-text display.
func stripSimpleHTML(s string) string {
	// Very naive tag stripper — sufficient for basic HTML bodies.
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<p>", "\n")
	s = strings.ReplaceAll(s, "</p>", "")
	s = strings.ReplaceAll(s, "<div>", "\n")
	s = strings.ReplaceAll(s, "</div>", "")
	// Remove remaining tags with a simple regex-like approach.
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func normalizeReplySubject(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(strings.ToLower(s), "re:") {
		s = strings.TrimSpace(s[3:])
	}
	if s == "" {
		return "Re: "
	}
	return "Re: " + s
}

func normalizeForwardSubject(s string) string {
	s = strings.TrimSpace(s)
	for {
		lower := strings.ToLower(s)
		if strings.HasPrefix(lower, "fwd:") || strings.HasPrefix(lower, "fw:") {
			if strings.HasPrefix(lower, "fwd:") {
				s = strings.TrimSpace(s[4:])
			} else {
				s = strings.TrimSpace(s[3:])
			}
			continue
		}
		break
	}
	if s == "" {
		return "Fwd: "
	}
	return "Fwd: " + s
}
