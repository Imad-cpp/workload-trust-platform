package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("WTP_LISTEN_ADDR", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected DATABASE_URL validation error")
	}
}

func TestLoadUsesLoopbackDefault(t *testing.T) {
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
	cfg := Config{ListenAddr: "0.0.0.0:8080", DatabaseURL: "postgres://example"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-loopback bind to be rejected")
	}
}

func TestValidateAcceptsIPv6Loopback(t *testing.T) {
	cfg := Config{ListenAddr: "[::1]:8080", DatabaseURL: "postgres://example"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
