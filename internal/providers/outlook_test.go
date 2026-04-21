package providers

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestOutlookEmbeddedClientID(t *testing.T) {
	t.Setenv(envOutlookClientID, "")

	o := NewOutlook()
	if err := o.ensureClientID(); err != nil {
		t.Fatal(err)
	}
	if o.oauthConfig.ClientID != defaultOutlookClientID {
		t.Fatalf("client_id: got %q", o.oauthConfig.ClientID)
	}
}

func TestOutlookEnvClientIDOverride(t *testing.T) {
	t.Setenv(envOutlookClientID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	o := NewOutlook()
	if err := o.ensureClientID(); err != nil {
		t.Fatal(err)
	}
	if o.oauthConfig.ClientID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("client_id: got %q", o.oauthConfig.ClientID)
	}
}

func TestOutlookOAuthEndpointAuthStyle(t *testing.T) {
	ep := outlookOAuthEndpoint()
	if ep.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("AuthStyle: got %v want InParams", ep.AuthStyle)
	}
}
