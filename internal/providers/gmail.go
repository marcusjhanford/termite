package providers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	gmailIMAPHost = "imap.gmail.com"
	gmailIMAPPort = 993
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPPort = 587

	gmailScope              = "https://mail.google.com/"
	gmailUserInfoEmailScope = "https://www.googleapis.com/auth/userinfo.email"
	gmailUserInfoURL        = "https://www.googleapis.com/oauth2/v2/userinfo"
	gmailAuthURL            = "https://accounts.google.com/o/oauth2/v2/auth"
	gmailTokenURL           = "https://oauth2.googleapis.com/token"
	// Loopback redirect for native OAuth. For Google client type "Desktop", the Cloud
	// Console often does not offer an "Authorized redirect URIs" list like "Web
	// application" clients—that is normal: loopback redirect URIs (localhost or
	// 127.0.0.1 with a port) are still valid for Desktop apps per Google's native-app
	// OAuth docs and loopback migration guide.
	gmailRedirectURL = "http://localhost:8765/callback"

	// defaultGmailClientID is Termite's Google OAuth client id (not secret). Prefer
	// registering it as type Desktop in Cloud Console with redirect gmailRedirectURL
	// so PKCE works without client_secret. If Google still requires a secret for this
	// client, official builds inject ReleaseGmailOAuthClientSecret via -ldflags (see below).
	defaultGmailClientID = "537656282375-6tf8ppher44dtavns3bl2tivuvegk34m.apps.googleusercontent.com"

	keyringService = "termite"

	// Developer overrides (optional): use alternate OAuth clients without rebuilding.
	envGmailCredentialsJSON = "TERMITE_GMAIL_CREDENTIALS_JSON"
	envGmailClientID      = "TERMITE_GMAIL_CLIENT_ID"
	envGmailClientSecret  = "TERMITE_GMAIL_CLIENT_SECRET"
)

// ReleaseGmailOAuthClientSecret is optional. Official release binaries inject the
// Google OAuth client_secret via -ldflags (must be exported for the Go linker).
// Local development: use environment variable TERMITE_GMAIL_CLIENT_SECRET instead.
//
//	go build -ldflags "-X github.com/termite-mail/termite/internal/providers.ReleaseGmailOAuthClientSecret=YOUR_SECRET"
//
// Leave empty in source; CI sets this only when producing official artifacts.
var ReleaseGmailOAuthClientSecret string

// Gmail implements the Provider interface for Google Mail using OAuth2 with PKCE.
type Gmail struct {
	oauthConfig *oauth2.Config
}

// NewGmail creates a new Gmail provider.
func NewGmail() *Gmail {
	g := &Gmail{
		oauthConfig: &oauth2.Config{
			Scopes: []string{gmailScope, gmailUserInfoEmailScope},
			Endpoint: googleTokenEndpoint(),
			RedirectURL: gmailRedirectURL,
		},
	}
	_ = g.ensureClientID()
	return g
}

func googleTokenEndpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  gmailAuthURL,
		TokenURL: gmailTokenURL,
	}
}

// SetClientID sets the OAuth2 client ID in-memory (e.g. from a local config UI).
// Developer env vars (see envGmail*) still override when set.
func (g *Gmail) SetClientID(clientID string) {
	g.oauthConfig.ClientID = clientID
	_ = g.ensureClientID()
}

func (g *Gmail) Name() string     { return "gmail" }
func (g *Gmail) IMAPHost() string { return gmailIMAPHost }
func (g *Gmail) IMAPPort() int    { return gmailIMAPPort }
func (g *Gmail) IMAPTLS() bool    { return true }
func (g *Gmail) SMTPHost() string { return gmailSMTPHost }
func (g *Gmail) SMTPPort() int    { return gmailSMTPPort }
func (g *Gmail) SMTPTLS() bool    { return true }

// GetCredentials retrieves a stored refresh token from the OS keyring,
// exchanges it for a fresh access token, and returns XOAUTH2 credentials.
func (g *Gmail) GetCredentials(accountID string) (Credentials, error) {
	if err := g.ensureClientID(); err != nil {
		return Credentials{}, err
	}

	refreshToken, err := keyring.Get(keyringService, accountID+"/gmail/refresh_token")
	if err != nil {
		return Credentials{}, fmt.Errorf("no stored credentials for %s: %w", accountID, err)
	}

	fresh, err := g.postGoogleToken(context.Background(), refreshTokenForm(g, refreshToken))
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Persist updated refresh token if it rotated.
	if fresh.RefreshToken != "" && fresh.RefreshToken != refreshToken {
		_ = keyring.Set(keyringService, accountID+"/gmail/refresh_token", fresh.RefreshToken)
	}

	return Credentials{
		Username:    accountID,
		AccessToken: fresh.AccessToken,
		AuthMethod:  "XOAUTH2",
	}, nil
}

func refreshTokenForm(g *Gmail, refreshToken string) url.Values {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {g.oauthConfig.ClientID},
	}
	setGoogleClientSecret(form, g.oauthConfig.ClientSecret)
	return form
}

func authCodeForm(g *Gmail, code, codeVerifier string) url.Values {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {g.oauthConfig.RedirectURL},
		"client_id":     {g.oauthConfig.ClientID},
		"code_verifier": {codeVerifier},
	}
	setGoogleClientSecret(form, g.oauthConfig.ClientSecret)
	return form
}

// setGoogleClientSecret adds client_secret only when non-empty.
// Google's token endpoint rejects client_secret= (empty) with "client_secret is missing";
// Desktop / installed + PKCE clients must omit the parameter entirely (see Google native-app docs).
func setGoogleClientSecret(form url.Values, secret string) {
	if secret != "" {
		form.Set("client_secret", secret)
	}
}

// postGoogleToken POSTs form to Google's token URL and parses the JSON token response.
func (g *Gmail) postGoogleToken(ctx context.Context, form url.Values) (*oauth2.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.oauthConfig.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var wire struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Err          string `json:"error"`
		ErrDesc      string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("token response: %w", err)
	}
	if wire.Err != "" {
		var err error
		if wire.ErrDesc != "" {
			err = fmt.Errorf("oauth2: %q %q", wire.Err, wire.ErrDesc)
		} else {
			err = fmt.Errorf("oauth2: %q", wire.Err)
		}
		return nil, explainGoogleClientSecretErr(g, err, wire.Err, wire.ErrDesc)
	}
	if wire.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	tok := &oauth2.Token{
		AccessToken:  wire.AccessToken,
		TokenType:    wire.TokenType,
		RefreshToken: wire.RefreshToken,
	}
	if wire.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(wire.ExpiresIn) * time.Second)
	}
	return tok, nil
}

func parseGoogleUserInfoEmailJSON(body []byte) (string, error) {
	var u struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return "", err
	}
	return strings.TrimSpace(u.Email), nil
}

// fetchGoogleUserPrimaryEmail returns the Google account email for this access token.
func fetchGoogleUserPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	if accessToken == "" {
		return "", fmt.Errorf("missing access token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gmailUserInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo: HTTP %d", resp.StatusCode)
	}
	return parseGoogleUserInfoEmailJSON(body)
}

// RunAuthFlow opens the OAuth2 consent flow using PKCE, starts a local HTTP
// server on port 8765, and waits for the redirect callback.
func (g *Gmail) RunAuthFlow(accountID string) (Credentials, error) {
	if err := g.ensureClientID(); err != nil {
		return Credentials{}, err
	}

	// Generate PKCE code verifier (43-128 chars, base64url-encoded random bytes).
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Derive code challenge (SHA-256, base64url, no padding).
	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	// Generate state parameter.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Build auth URL with PKCE params.
	authURL := g.oauthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("login_hint", accountID),
		oauth2.AccessTypeOffline,
	)

	if err := openBrowser(authURL); err != nil {
		return Credentials{}, fmt.Errorf("failed to open browser: %w", err)
	}

	// Start local server to receive the callback.
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
		fmt.Fprintln(w, "Authentication successful! You can close this window.")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", "127.0.0.1:8765")
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to listen on port 8765: %w", err)
	}
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	// Wait for the authorization code.
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return Credentials{}, err
	}

	// Manual token POST: omit empty client_secret (Google rejects client_secret=).
	// Termite's default Google client is public Desktop + PKCE; optional env secret
	// is for developer overrides only (e.g. alternate OAuth client types).
	token, err := g.postGoogleToken(context.Background(), authCodeForm(g, code, codeVerifier))
	if err != nil {
		return Credentials{}, fmt.Errorf("token exchange failed: %w", err)
	}

	ctxUser, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resolved, errUI := fetchGoogleUserPrimaryEmail(ctxUser, token.AccessToken)
	if errUI != nil || resolved == "" {
		resolved = accountID
	}

	// Keyring and IMAP SASL identity use the Google-confirmed mailbox when available.
	if token.RefreshToken != "" {
		if err := keyring.Set(keyringService, resolved+"/gmail/refresh_token", token.RefreshToken); err != nil {
			return Credentials{}, fmt.Errorf("failed to store refresh token in keyring: %w", err)
		}
	}

	return Credentials{
		Username:      resolved,
		AccessToken:   token.AccessToken,
		AuthMethod:    "XOAUTH2",
		IdentityEmail: resolved,
	}, nil
}

// RefreshToken re-fetches a fresh access token using the stored refresh token.
func (g *Gmail) RefreshToken(accountID string) (Credentials, error) {
	return g.GetCredentials(accountID)
}

// googleOAuthClientJSON matches the downloaded credentials file from Google Cloud
// Console (installed or web client).
type googleOAuthClientJSON struct {
	Web       *googleOAuthClientSection `json:"web"`
	Installed *googleOAuthClientSection `json:"installed"`
}

type googleOAuthClientSection struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
}

func (g *Gmail) mergeGoogleOAuthJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f googleOAuthClientJSON
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	sec := f.Installed
	if sec == nil {
		sec = f.Web
	}
	if sec == nil {
		return fmt.Errorf("JSON must contain \"installed\" or \"web\" OAuth client section")
	}
	if sec.ClientID == "" {
		return fmt.Errorf("JSON missing client_id")
	}
	g.oauthConfig.ClientID = sec.ClientID
	g.oauthConfig.ClientSecret = sec.ClientSecret
	return nil
}

// ensureClientID sets OAuth client id/secret: optional developer JSON path or env
// overrides, then Termite's default public Desktop client id when still unset.
func (g *Gmail) ensureClientID() error {
	if path := os.Getenv(envGmailCredentialsJSON); path != "" {
		if err := g.mergeGoogleOAuthJSONFile(path); err != nil {
			return fmt.Errorf("%s: %w", envGmailCredentialsJSON, err)
		}
	}
	if id := os.Getenv(envGmailClientID); id != "" {
		g.oauthConfig.ClientID = id
	}
	if secret := os.Getenv(envGmailClientSecret); secret != "" {
		g.oauthConfig.ClientSecret = secret
	} else if g.oauthConfig.ClientSecret == "" && ReleaseGmailOAuthClientSecret != "" {
		g.oauthConfig.ClientSecret = ReleaseGmailOAuthClientSecret
	}
	if g.oauthConfig.ClientID == "" {
		g.oauthConfig.ClientID = defaultGmailClientID
	}

	return nil
}

// explainGoogleClientSecretErr adds context when Google rejects the token request
// for missing client_secret even with PKCE (common when the OAuth client is type
// "Web application" or when the Console still issued a secret for "Desktop").
func explainGoogleClientSecretErr(g *Gmail, err error, code, desc string) error {
	if err == nil || g.oauthConfig.ClientSecret != "" {
		return err
	}
	if code != "invalid_request" || !strings.Contains(strings.ToLower(desc), "client_secret") {
		return err
	}
	return fmt.Errorf("%w — Google still requires a client_secret for this OAuth client. Use an official Termite build, set %s for local testing, or register a Desktop client with redirect %s (PKCE-only) and update the embedded client id",
		err, envGmailClientSecret, gmailRedirectURL)
}

// openBrowser launches the user's default browser to the given URL.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
