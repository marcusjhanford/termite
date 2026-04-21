package providers

import "testing"

func TestParseGoogleUserInfoEmailJSON(t *testing.T) {
	email, err := parseGoogleUserInfoEmailJSON([]byte(`{"email":"real.user@gmail.com","verified_email":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if email != "real.user@gmail.com" {
		t.Fatalf("email: %q", email)
	}
}

func TestParseGoogleUserInfoEmailJSONWhitespace(t *testing.T) {
	email, err := parseGoogleUserInfoEmailJSON([]byte(`{"email":"  x@y.co  "}`))
	if err != nil {
		t.Fatal(err)
	}
	if email != "x@y.co" {
		t.Fatalf("email: %q", email)
	}
}
