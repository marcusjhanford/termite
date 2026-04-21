package providers

import "fmt"

const (
	fastmailIMAPHost = "imap.fastmail.com"
	fastmailIMAPPort = 993
	fastmailSMTPHost = "smtp.fastmail.com"
	fastmailSMTPPort = 587
)

// Fastmail implements the Provider interface for Fastmail using app passwords.
type Fastmail struct{}

// NewFastmail creates a new Fastmail provider.
func NewFastmail() *Fastmail {
	return &Fastmail{}
}

func (f *Fastmail) Name() string     { return "fastmail" }
func (f *Fastmail) IMAPHost() string { return fastmailIMAPHost }
func (f *Fastmail) IMAPPort() int    { return fastmailIMAPPort }
func (f *Fastmail) IMAPTLS() bool    { return true }
func (f *Fastmail) SMTPHost() string { return fastmailSMTPHost }
func (f *Fastmail) SMTPPort() int    { return fastmailSMTPPort }
func (f *Fastmail) SMTPTLS() bool    { return true }

func (f *Fastmail) GetCredentials(accountID string) (Credentials, error) {
	return Credentials{}, fmt.Errorf("fastmail: GetCredentials not yet implemented")
}

func (f *Fastmail) RunAuthFlow(accountID string) (Credentials, error) {
	return Credentials{}, fmt.Errorf("fastmail: RunAuthFlow not yet implemented")
}

func (f *Fastmail) RefreshToken(accountID string) (Credentials, error) {
	return Credentials{}, fmt.Errorf("fastmail: RefreshToken not yet implemented")
}
