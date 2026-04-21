package composepage

import (
	"encoding/json"
	"strings"

	"github.com/termite-mail/termite/internal/db"
)

// FromThreadDraft builds a compose model with recipients and subject from the latest
// message in a thread. The body is left empty so the user types above the visible message.
// latest may be nil (caller keeps defaults).
func FromThreadDraft(mode, threadID string, latest *db.Message, accountEmails []string) Model {
	m := NewWithMode(mode, threadID)
	if latest == nil {
		return m
	}

	self := normalizeSet(accountEmails)

	switch mode {
	case "reply":
		m.to = strings.TrimSpace(latest.FromAddr)
		m.cc = ""
		m.subject = normalizeReplySubject(latest.Subject)
		m.body = ""
		m.activeField = fieldBody
	case "replyall":
		to, cc := replyAllRecipients(latest, self)
		m.to = to
		m.cc = cc
		m.subject = normalizeReplySubject(latest.Subject)
		m.body = ""
		m.activeField = fieldBody
	case "forward":
		m.to = ""
		m.cc = ""
		m.subject = normalizeForwardSubject(latest.Subject)
		m.body = ""
		m.activeField = fieldTo
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
