package providers

import "fmt"

// Credentials holds auth data for an email provider.
type Credentials struct {
	Username    string
	Password    string // app password or OAuth access token
	AccessToken string // for OAuth providers
	AuthMethod  string // "PLAIN", "XOAUTH2"
	// IdentityEmail is the provider-confirmed mailbox (e.g. from Google userinfo
	// after OAuth). Empty when unknown; use Username for IMAP SASL identity.
	IdentityEmail string
}

// Provider defines the interface all email providers must implement.
type Provider interface {
	Name() string
	IMAPHost() string
	IMAPPort() int
	IMAPTLS() bool
	SMTPHost() string
	SMTPPort() int
	SMTPTLS() bool
	GetCredentials(accountID string) (Credentials, error)
	RunAuthFlow(accountID string) (Credentials, error)
	RefreshToken(accountID string) (Credentials, error)
}

// NewProvider creates a Provider from a provider name string.
func NewProvider(name string) (Provider, error) {
	switch name {
	case "gmail":
		return NewGmail(), nil
	case "outlook":
		return NewOutlook(), nil
	case "fastmail":
		return NewFastmail(), nil
	case "generic":
		return NewGeneric("", 993, true, "", 587, true), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
