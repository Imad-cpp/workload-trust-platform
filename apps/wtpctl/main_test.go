package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const cliTestToken = "0123456789abcdef0123456789abcdef"

func TestValidateBaseURLRejectsNonLoopbackAndUnsafeForms(t *testing.T) {
	for _, raw := range []string{
		"https://127.0.0.1:8080",
		"http://0.0.0.0:8080",
		"http://192.0.2.10:8080",
		"http://example.test:8080",
		"http://user:pass@127.0.0.1:8080",
		"http://127.0.0.1:8080/admin",
		"http://127.0.0.1:8080?token=x",
		"http://127.0.0.1:8080#fragment",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := validateBaseURL(raw); err == nil {
				t.Fatalf("validateBaseURL(%q) unexpectedly succeeded", raw)
			}
		})
	}

	for _, raw := range []string{
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://[::1]:8080",
	} {
		if _, err := validateBaseURL(raw); err != nil {
			t.Fatalf("validateBaseURL(%q) error = %v", raw, err)
		}
	}
}

func TestStatusUsesBearerAndPrintsServerJSON(t *testing.T) {
	organizationID := "123e4567-e89b-12d3-a456-426614174000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/diagnostics" || r.URL.Query().Get("organization_id") != organizationID {
			t.Fatalf("unexpected request URL: %s", r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer "+cliTestToken {
			t.Fatalf("unexpected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"organization_id":"` + organizationID + `","workloads_total":2}}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        server.URL,
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	if err := run([]string{"status", "--organization-id", organizationID}, env, &stdout, &stderr); err != nil {
		t.Fatalf("run(status) error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"workloads_total":2`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), cliTestToken) || strings.Contains(stderr.String(), cliTestToken) {
		t.Fatal("CLI output leaked operator token")
	}
}

func TestHealthDoesNotSendBearerCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("health unexpectedly sent Authorization: %q", got)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        server.URL,
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	if err := run([]string{"health"}, env, &stdout, &stderr); err != nil {
		t.Fatalf("run(health) error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health output: %s", stdout.String())
	}
}

func TestAuthenticatedCommandRequiresStrongToken(t *testing.T) {
	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        "http://127.0.0.1:8080",
		"WTP_OPERATOR_TOKEN": "short",
	})
	err := run([]string{"status", "--organization-id", "123e4567-e89b-12d3-a456-426614174000"}, env, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Fatalf("unexpected token error: %v", err)
	}
}

func TestStatusRequiresOrganizationID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        "http://127.0.0.1:8080",
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	err := run([]string{"status"}, env, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "requires --organization-id") {
		t.Fatalf("unexpected organization error: %v", err)
	}
}

func TestAPIErrorDoesNotEchoResponseBodyOrToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"secret ` + cliTestToken + `"}}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        server.URL,
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	err := run([]string{"status", "--organization-id", "123e4567-e89b-12d3-a456-426614174000"}, env, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "HTTP 401 (unauthorized)") {
		t.Fatalf("unexpected API error: %v", err)
	}
	combined := stdout.String() + stderr.String() + err.Error()
	if strings.Contains(combined, cliTestToken) || strings.Contains(combined, "secret ") {
		t.Fatal("CLI error leaked response body or token")
	}
}

func TestRedirectIsNotFollowed(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        redirector.URL,
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	err := run([]string{"status", "--organization-id", "123e4567-e89b-12d3-a456-426614174000"}, env, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("unexpected redirect result: %v", err)
	}
	if targetCalls != 0 {
		t.Fatalf("CLI followed redirect %d time(s)", targetCalls)
	}
}

func envMap(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
