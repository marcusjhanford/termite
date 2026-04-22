package engine

import (
	"fmt"
	"strings"
)

// SendComposeSMTP sends a compose draft via SMTP using the account's provider settings.
// attachments must be empty for now (unsupported).
func SendComposeSMTP(acct Account, from string, to, cc, bcc, subject, body string, attachments []string) error {
	if len(strings.TrimSpace(from)) == 0 {
		return fmt.Errorf("missing from address")
	}
	if len(attachments) > 0 {
		return fmt.Errorf("attachments are not supported yet; remove attachments to send")
	}
	toList := parseAddressList(to)
	ccList := parseAddressList(cc)
	bccList := parseAddressList(bcc)
	if len(toList) == 0 {
		return fmt.Errorf("add at least one To recipient")
	}

	creds, err := acct.Provider.GetCredentials(acct.Config.ID)
	if err != nil {
		return fmt.Errorf("credentials: %w", err)
	}

	raw := buildRFC822(from, toList, ccList, subject, body)
	rcpt := append(append(append([]string{}, toList...), ccList...), bccList...)

	var sender SMTPSender
	return sender.Send(
		acct.Provider.SMTPHost(),
		acct.Provider.SMTPPort(),
		acct.Provider.SMTPTLS(),
		creds,
		from,
		rcpt,
		raw,
	)
}

func parseAddressList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildRFC822(from string, to, cc []string, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(to, ", "))
	if len(cc) > 0 {
		fmt.Fprintf(&b, "Cc: %s\r\n", strings.Join(cc, ", "))
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&b, "\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
