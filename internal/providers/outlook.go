package providers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	outlookIMAPHost = "outlook.office365.com"
	outlookIMAPPort = 993
	outlookSMTPHost = "smtp.office365.com"
	outlookSMTPPort = 587

	// Same loopback callback as Gmail; register this exact URI on the Termite
	// Entra "public client" app (Desktop / native redirect).
	outlookRedirectURL = "http://localhost:8765/callback"

	// defaultOutlookClientID is the Termite Entra application (client) ID for a
	// multi-tenant public client. Replace with the real ID from Azure Portal
	// after registration (not a secret).
	defaultOutlookClientID = "c0d8a5c4-5f3e-4a1b-9c2d-8e7f6a5b4c3d"

	envOutlookClientID = "TERMITE_OUTLOOK_CLIENT_ID"
)

func outlookOAuthEndpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:   "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		AuthStyle: oauth2.AuthStyleInParams,
	}
}

func outlookClientID() string {
	if id := os.Getenv(envOutlookClientID); id != "" {
		return id
	}
	return defaultOutlookClientID
}

// Outlook implements Microsoft 365 / Outlook.com via OAuth2 (PKCE, public client)
// and IMAP XOAUTH2.
type Outlook struct {
	oauthConfig *oauth2.Config
}

// NewOutlook creates a new Outlook provider with Termite's embedded public client id.
func NewOutlook() *Outlook {
	o := &Outlook{
		oauthConfig: &oauth2.Config{
			ClientID:     outlookClientID(),
			ClientSecret: "",
			RedirectURL:  outlookRedirectURL,
			Scopes: []string{
				"offline_access",
				"https://outlook.office.com/IMAP.AccessAsUser.All",
			},
			Endpoint: outlookOAuthEndpoint(),
		},
	}
	_ = o.ensureClientID()
	return o
}

func (o *Outlook) ensureClientID() error {
	o.oauthConfig.ClientID = outlookClientID()
	return nil
}

func (o *Outlook) Name() string     { return "outlook" }
func (o *Outlook) IMAPHost() string { return outlookIMAPHost }
func (o *Outlook) IMAPPort() int    { return outlookIMAPPort }
func (o *Outlook) IMAPTLS() bool    { return true }
func (o *Outlook) SMTPHost() string { return outlookSMTPHost }
func (o *Outlook) SMTPPort() int    { return outlookSMTPPort }
func (o *Outlook) SMTPTLS() bool    { return true }

// GetCredentials loads a refresh token from the keyring and refreshes the access token.
func (o *Outlook) GetCredentials(accountID string) (Credentials, error) {
	if err := o.ensureClientID(); err != nil {
		return Credentials{}, err
	}

	refreshToken, err := keyring.Get(keyringService, accountID+"/outlook/refresh_token")
	if err != nil {
		return Credentials{}, fmt.Errorf("no stored credentials for %s: %w", accountID, err)
	}

	ctx := context.Background()
	src := o.oauthConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to refresh token: %w", err)
	}

	if tok.RefreshToken != "" && tok.RefreshToken != refreshToken {
		_ = keyring.Set(keyringService, accountID+"/outlook/refresh_token", tok.RefreshToken)
	}

	return Credentials{
		Username:    accountID,
		AccessToken: tok.AccessToken,
		AuthMethod:  "XOAUTH2",
	}, nil
}

// RunAuthFlow opens the Microsoft login page (PKCE), receives the callback on localhost,
// exchanges the code, and stores the refresh token in the OS keyring.
func (o *Outlook) RunAuthFlow(accountID string) (Credentials, error) {
	if err := o.ensureClientID(); err != nil {
		return Credentials{}, err
	}

	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	authURL := o.oauthConfig.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("login_hint", accountID),
		oauth2.SetAuthURLParam("response_mode", "query"),
	)

	if err := openBrowser(authURL); err != nil {
		return Credentials{}, fmt.Errorf("failed to open browser: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			desc := r.URL.Query().Get("error_description")
			if desc != "" {
				errCh <- fmt.Errorf("oauth error: %s (%s)", errParam, desc)
			} else {
				errCh <- fmt.Errorf("oauth error: %s", errParam)
			}
			http.Error(w, errParam, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "no code", http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprintln(w, "Authentication successful! You can close this window.")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", "127.0.0.1:8765")
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to listen on port 8765: %w", err)
	}
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		_ = server.Close()
		_ = listener.Close()
	}()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return Credentials{}, err
	case <-time.After(10 * time.Minute):
		return Credentials{}, fmt.Errorf("authentication timed out")
	}

	ctx := context.Background()
	token, err := o.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return Credentials{}, fmt.Errorf("token exchange failed: %w", err)
	}

	if token.RefreshToken != "" {
		if err := keyring.Set(keyringService, accountID+"/outlook/refresh_token", token.RefreshToken); err != nil {
			return Credentials{}, fmt.Errorf("failed to store refresh token in keyring: %w", err)
		}
	}

	return Credentials{
		Username:    accountID,
		AccessToken: token.AccessToken,
		AuthMethod:  "XOAUTH2",
	}, nil
}

// RefreshToken returns fresh credentials using the stored refresh token.
func (o *Outlook) RefreshToken(accountID string) (Credentials, error) {
	return o.GetCredentials(accountID)
}
