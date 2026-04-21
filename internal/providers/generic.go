package providers

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

// Generic implements the Provider interface for arbitrary IMAP/SMTP servers.
// The caller supplies host, port, and TLS settings. Passwords are stored in
// the OS keyring under the "termite" service.
type Generic struct {
	imapHost string
	imapPort int
	imapTLS  bool
	smtpHost string
	smtpPort int
	smtpTLS  bool
}

// NewGeneric creates a Generic provider with the given server settings.
func NewGeneric(imapHost string, imapPort int, imapTLS bool, smtpHost string, smtpPort int, smtpTLS bool) *Generic {
	return &Generic{
		imapHost: imapHost,
		imapPort: imapPort,
		imapTLS:  imapTLS,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		smtpTLS:  smtpTLS,
	}
}

func (g *Generic) Name() string     { return "generic" }
func (g *Generic) IMAPHost() string { return g.imapHost }
func (g *Generic) IMAPPort() int    { return g.imapPort }
func (g *Generic) IMAPTLS() bool    { return g.imapTLS }
func (g *Generic) SMTPHost() string { return g.smtpHost }
func (g *Generic) SMTPPort() int    { return g.smtpPort }
func (g *Generic) SMTPTLS() bool    { return g.smtpTLS }

// GetCredentials retrieves the stored password for the given account from the
// OS keyring and returns PLAIN auth credentials.
func (g *Generic) GetCredentials(accountID string) (Credentials, error) {
	password, err := keyring.Get(keyringService, accountID+"/generic/password")
	if err != nil {
		return Credentials{}, fmt.Errorf("no stored credentials for %s: %w", accountID, err)
	}

	return Credentials{
		Username:   accountID,
		Password:   password,
		AuthMethod: "PLAIN",
	}, nil
}

// RunAuthFlow prompts for a password (placeholder) and stores it in the OS keyring.
func (g *Generic) RunAuthFlow(accountID string) (Credentials, error) {
	// In a real implementation this would prompt the user for their password
	// via the TUI. For now, return an error indicating that the caller must
	// supply a password and store it with StorePassword.
	return Credentials{}, fmt.Errorf("generic: RunAuthFlow requires a password; use StorePassword to save credentials")
}

// StorePassword saves a password to the OS keyring for later retrieval via
// GetCredentials.
func (g *Generic) StorePassword(accountID, password string) error {
	return keyring.Set(keyringService, accountID+"/generic/password", password)
}

// RefreshToken is a no-op for password-based providers. It simply returns
// the stored credentials.
func (g *Generic) RefreshToken(accountID string) (Credentials, error) {
	return g.GetCredentials(accountID)
}
