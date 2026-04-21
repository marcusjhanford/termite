package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureClientIDLoadsInstalledCredentialsJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.json")
	body := `{"installed":{"client_id":"fromfile.apps.googleusercontent.com","client_secret":"fromfilesecret","redirect_uris":["http://localhost:8765/callback"]}}`
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", p)
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	g := NewGmail()
	if g.oauthConfig.ClientID != "fromfile.apps.googleusercontent.com" {
		t.Fatalf("client_id: got %q", g.oauthConfig.ClientID)
	}
	if g.oauthConfig.ClientSecret != "fromfilesecret" {
		t.Fatalf("client_secret: got %q", g.oauthConfig.ClientSecret)
	}
}

func TestEnsureClientIDWebSectionCredentialsJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "web.json")
	body := `{"web":{"client_id":"webid.apps.googleusercontent.com","client_secret":"websec"}}`
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", p)
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	g := NewGmail()
	if g.oauthConfig.ClientID != "webid.apps.googleusercontent.com" {
		t.Fatalf("client_id: got %q", g.oauthConfig.ClientID)
	}
	if g.oauthConfig.ClientSecret != "websec" {
		t.Fatalf("client_secret: got %q", g.oauthConfig.ClientSecret)
	}
}

func TestEnsureClientIDEnvClientIDOverridesFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.json")
	body := `{"installed":{"client_id":"fromfile.apps.googleusercontent.com","client_secret":"fromfilesecret"}}`
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", p)
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "fromenv.apps.googleusercontent.com")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	g := NewGmail()
	if g.oauthConfig.ClientID != "fromenv.apps.googleusercontent.com" {
		t.Fatalf("client_id: got %q", g.oauthConfig.ClientID)
	}
	if g.oauthConfig.ClientSecret != "fromfilesecret" {
		t.Fatalf("client_secret should stay from file: got %q", g.oauthConfig.ClientSecret)
	}
}

func TestMergeGoogleOAuthJSONFileInvalid(t *testing.T) {
	g := NewGmail()
	err := g.mergeGoogleOAuthJSONFile("/nonexistent/path/credentials.json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureClientIDUsesLdflagsDefaultSecret(t *testing.T) {
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	prev := ReleaseGmailOAuthClientSecret
	ReleaseGmailOAuthClientSecret = "release-injected-secret"
	t.Cleanup(func() { ReleaseGmailOAuthClientSecret = prev })

	g := NewGmail()
	if err := g.ensureClientID(); err != nil {
		t.Fatal(err)
	}
	if g.oauthConfig.ClientSecret != "release-injected-secret" {
		t.Fatalf("client_secret: got %q", g.oauthConfig.ClientSecret)
	}
}

func TestEnsureClientIDDefaultEmbeddedWhenNoOverrides(t *testing.T) {
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	g := NewGmail()
	if err := g.ensureClientID(); err != nil {
		t.Fatal(err)
	}
	if g.oauthConfig.ClientID != defaultGmailClientID {
		t.Fatalf("client_id: want default, got %q", g.oauthConfig.ClientID)
	}
	if g.oauthConfig.ClientSecret != "" {
		t.Fatalf("client_secret: want empty by default, got %q", g.oauthConfig.ClientSecret)
	}
}

func TestEnsureClientIDAllowsSetClientIDWithoutEnv(t *testing.T) {
	t.Setenv("TERMITE_GMAIL_CREDENTIALS_JSON", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_ID", "")
	t.Setenv("TERMITE_GMAIL_CLIENT_SECRET", "")

	g := NewGmail()
	g.SetClientID("local.apps.googleusercontent.com")
	if err := g.ensureClientID(); err != nil {
		t.Fatalf("SetClientID should satisfy ensureClientID: %v", err)
	}
	if g.oauthConfig.ClientID != "local.apps.googleusercontent.com" {
		t.Fatalf("client id: %q", g.oauthConfig.ClientID)
	}
}
