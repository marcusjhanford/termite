package providers

import (
	"net/url"
	"strings"
	"testing"
)

func TestURLEncodeIncludesEmptyClientSecret(t *testing.T) {
	v := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"abc"},
		"redirect_uri":  {"http://localhost:8765/callback"},
		"code_verifier": {"verifier"},
		"client_secret": {""},
	}
	v.Set("client_id", "myid")
	s := v.Encode()
	if !strings.Contains(s, "client_secret=") {
		t.Fatalf("expected client_secret= in encoded form, got %q", s)
	}
}

func TestAuthCodeFormOmitsEmptyClientSecret(t *testing.T) {
	g := NewGmail()
	f := authCodeForm(g, "abc", "verifier")
	if _, ok := f["client_secret"]; ok {
		t.Fatalf("public client must omit client_secret key, got keys=%v", keysOf(f))
	}
	if strings.Contains(f.Encode(), "client_secret") {
		t.Fatalf("encoded form must not mention client_secret when unset")
	}
	if f.Get("client_id") != defaultGmailClientID {
		t.Fatalf("client_id: got %q", f.Get("client_id"))
	}
}

func keysOf(v url.Values) []string {
	out := make([]string, 0, len(v))
	for k := range v {
		out = append(out, k)
	}
	return out
}
