package config

import "testing"

const testOperatorToken = "0123456789abcdef0123456789abcdef"

func setValidOperatorEnv(t *testing.T) {
	t.Helper()
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", testOperatorToken)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setValidOperatorEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("WTP_LISTEN_ADDR", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected DATABASE_URL validation error")
	}
}

func TestLoadRequiresOperatorIdentity(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_OPERATOR_ID", "")
	t.Setenv("WTP_OPERATOR_TOKEN", testOperatorToken)

	if _, err := Load(); err == nil {
		t.Fatal("expected WTP_OPERATOR_ID validation error")
	}
}

func TestLoadRequiresOperatorToken(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected WTP_OPERATOR_TOKEN validation error")
	}
}

func TestLoadUsesLoopbackDefault(t *testing.T) {
	setValidOperatorEnv(t)
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_LISTEN_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" {
		t.Fatalf("ListenAddr = %q, want loopback default", cfg.ListenAddr)
	}
}

func TestValidateRejectsNonLoopbackBind(t *testing.T) {
	cfg := Config{
		ListenAddr:    "0.0.0.0:8080",
		DatabaseURL:   "postgres://example",
		OperatorID:    "local-admin",
		OperatorToken: testOperatorToken,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-loopback bind to be rejected")
	}
}

func TestValidateAcceptsIPv6Loopback(t *testing.T) {
	cfg := Config{
		ListenAddr:    "[::1]:8080",
		DatabaseURL:   "postgres://example",
		OperatorID:    "local-admin",
		OperatorToken: testOperatorToken,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
