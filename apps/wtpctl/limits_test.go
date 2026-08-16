package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOversizedManagementResponseIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"` + strings.Repeat("x", maxResponseBytes) + `"}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	env := envMap(map[string]string{
		"WTP_API_URL":        server.URL,
		"WTP_OPERATOR_TOKEN": cliTestToken,
	})
	err := run([]string{"status", "--organization-id", "123e4567-e89b-12d3-a456-426614174000"}, env, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exceeded 1 MiB limit") {
		t.Fatalf("unexpected oversized response result: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("oversized response wrote output: %d bytes", stdout.Len())
	}
}
