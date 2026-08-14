package operatorauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testToken = "0123456789abcdef0123456789abcdef"

func TestNewStaticBearerRejectsShortToken(t *testing.T) {
	if _, err := NewStaticBearer("too-short", "local-admin"); err == nil {
		t.Fatal("expected short token to be rejected")
	}
}

func TestStaticBearerAuthenticatesExactToken(t *testing.T) {
	auth, err := NewStaticBearer(testToken, "local-admin")
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/workloads", nil)
	r.Header.Set("Authorization", "Bearer "+testToken)
	principal, ok := auth.Authenticate(r)
	if !ok {
		t.Fatal("expected valid bearer token to authenticate")
	}
	if principal.ActorType != "operator" || principal.ActorID != "local-admin" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestStaticBearerRejectsMalformedOrWrongCredentials(t *testing.T) {
	auth, err := NewStaticBearer(testToken, "local-admin")
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}

	for _, header := range []string{
		"",
		"Basic " + testToken,
		"Bearer",
		"Bearer wrong-token-that-is-definitely-long-enough",
		"Bearer " + testToken + " extra",
	} {
		r := httptest.NewRequest(http.MethodGet, "/v1/workloads", nil)
		r.Header.Set("Authorization", header)
		if principal, ok := auth.Authenticate(r); ok {
			t.Fatalf("header %q unexpectedly authenticated as %#v", header, principal)
		}
	}
}
