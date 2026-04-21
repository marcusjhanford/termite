package engine

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/termite-mail/termite/internal/providers"
)

// SMTPSender sends email via SMTP.
type SMTPSender struct{}

// Send delivers an email message via SMTP with the given credentials.
// It supports STARTTLS for TLS-enabled connections and handles both
// XOAUTH2 and PLAIN authentication methods.
func (s *SMTPSender) Send(host string, port int, tlsEnabled bool, creds providers.Credentials, from string, to []string, body []byte) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Connect to the SMTP server.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: failed to connect to %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp: failed to create client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS via STARTTLS if enabled.
	if tlsEnabled {
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp: STARTTLS failed: %w", err)
		}
	}

	// Authenticate.
	var auth smtp.Auth
	switch creds.AuthMethod {
	case "XOAUTH2":
		auth = &xoauth2SMTPAuth{
			username:    creds.Username,
			accessToken: creds.AccessToken,
		}
	default:
		// PLAIN auth via net/smtp.
		password := creds.Password
		if password == "" {
			password = creds.AccessToken
		}
		auth = smtp.PlainAuth("", creds.Username, password, host)
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp: authentication failed: %w", err)
	}

	// Set the sender.
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp: MAIL FROM failed: %w", err)
	}

	// Set the recipients.
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: RCPT TO <%s> failed: %w", rcpt, err)
		}
	}

	// Write the message body.
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA command failed: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp: failed to write message body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: failed to close data writer: %w", err)
	}

	// Quit the session.
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp: QUIT failed: %w", err)
	}

	return nil
}

// xoauth2SMTPAuth implements smtp.Auth for the XOAUTH2 SASL mechanism.
// This is used for Gmail/Outlook SMTP authentication with OAuth2 tokens.
type xoauth2SMTPAuth struct {
	username    string
	accessToken string
}

func (a *xoauth2SMTPAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	resp := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.accessToken)
	return "XOAUTH2", []byte(resp), nil
}

func (a *xoauth2SMTPAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// Server is requesting more data, which indicates an error in XOAUTH2.
		// Send empty response per the protocol spec.
		return []byte{}, nil
	}
	return nil, nil
}
