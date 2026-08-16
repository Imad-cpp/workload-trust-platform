package config

import (
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
)

const testOperatorToken = "0123456789abcdef0123456789abcdef"

func setValidOperatorEnv(t *testing.T) {
	t.Helper()
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", testOperatorToken)
	t.Setenv("WTP_OPERATOR_ROLE", "viewer")
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
	t.Setenv("WTP_OPERATOR_ROLE", "viewer")

	if _, err := Load(); err == nil {
		t.Fatal("expected WTP_OPERATOR_ID validation error")
	}
}

func TestLoadRequiresOperatorToken(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", "")
	t.Setenv("WTP_OPERATOR_ROLE", "viewer")

	if _, err := Load(); err == nil {
		t.Fatal("expected WTP_OPERATOR_TOKEN validation error")
	}
}

func TestLoadDefaultsOperatorRoleToViewer(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", testOperatorToken)
	t.Setenv("WTP_OPERATOR_ROLE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OperatorRole != operatorauth.RoleViewer {
		t.Fatalf("OperatorRole = %q, want %q", cfg.OperatorRole, operatorauth.RoleViewer)
	}
}

func TestLoadRejectsUnknownOperatorRole(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_OPERATOR_ID", "local-admin")
	t.Setenv("WTP_OPERATOR_TOKEN", testOperatorToken)
	t.Setenv("WTP_OPERATOR_ROLE", "owner")

	if _, err := Load(); err == nil {
		t.Fatal("expected unsupported operator role to fail")
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
		OperatorRole:  operatorauth.RoleViewer,
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
		OperatorRole:  operatorauth.RoleViewer,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadReconcilerUsesLocalSPIRESocketDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_SPIRE_SERVER_SOCKET", "")
	cfg, err := LoadReconciler()
	if err != nil {
		t.Fatalf("LoadReconciler() error = %v", err)
	}
	if cfg.SPIREServerSocket != "/tmp/workload-trust-lab/server.sock" {
		t.Fatalf("SPIREServerSocket = %q", cfg.SPIREServerSocket)
	}
}

func TestLoadReconcilerRejectsRelativeSocket(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WTP_SPIRE_SERVER_SOCKET", "relative/server.sock")
	if _, err := LoadReconciler(); err == nil {
		t.Fatal("expected relative SPIRE socket to be rejected")
	}
}
