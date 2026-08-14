package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
)

const (
	defaultListenAddr        = "127.0.0.1:8080"
	defaultSPIREServerSocket = "/tmp/workload-trust-lab/server.sock"
)

var operatorIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@:-]{2,127}$`)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	OperatorID    string
	OperatorToken string
	OperatorRole  operatorauth.Role
}

type ReconcilerConfig struct {
	DatabaseURL       string
	SPIREServerSocket string
}

func Load() (Config, error) {
	listenAddr := strings.TrimSpace(os.Getenv("WTP_LISTEN_ADDR"))
	if listenAddr == "" {
		listenAddr = defaultListenAddr
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	operatorID := strings.TrimSpace(os.Getenv("WTP_OPERATOR_ID"))
	if operatorID == "" {
		return Config{}, errors.New("WTP_OPERATOR_ID is required")
	}

	operatorToken := strings.TrimSpace(os.Getenv("WTP_OPERATOR_TOKEN"))
	if operatorToken == "" {
		return Config{}, errors.New("WTP_OPERATOR_TOKEN is required")
	}

	operatorRoleRaw := strings.TrimSpace(os.Getenv("WTP_OPERATOR_ROLE"))
	if operatorRoleRaw == "" {
		operatorRoleRaw = string(operatorauth.RoleViewer)
	}
	operatorRole, err := operatorauth.ParseRole(operatorRoleRaw)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ListenAddr:    listenAddr,
		DatabaseURL:   databaseURL,
		OperatorID:    operatorID,
		OperatorToken: operatorToken,
		OperatorRole:  operatorRole,
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadReconciler() (ReconcilerConfig, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return ReconcilerConfig{}, errors.New("DATABASE_URL is required")
	}
	socketPath := strings.TrimSpace(os.Getenv("WTP_SPIRE_SERVER_SOCKET"))
	if socketPath == "" {
		socketPath = defaultSPIREServerSocket
	}
	if !strings.HasPrefix(socketPath, "/") {
		return ReconcilerConfig{}, errors.New("WTP_SPIRE_SERVER_SOCKET must be an absolute path")
	}
	return ReconcilerConfig{DatabaseURL: databaseURL, SPIREServerSocket: socketPath}, nil
}

func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.ListenAddr)
	if err != nil {
		return fmt.Errorf("invalid WTP_LISTEN_ADDR: %w", err)
	}
	if port == "" {
		return errors.New("WTP_LISTEN_ADDR must include a port")
	}

	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return errors.New("WTP_LISTEN_ADDR must bind to a loopback address during the Phase 2 local operator API")
		}
	}
	if !operatorIDPattern.MatchString(c.OperatorID) {
		return errors.New("WTP_OPERATOR_ID has an invalid format")
	}
	if strings.TrimSpace(c.OperatorToken) == "" {
		return errors.New("WTP_OPERATOR_TOKEN is required")
	}
	if !c.OperatorRole.Valid() {
		return errors.New("WTP_OPERATOR_ROLE is invalid")
	}
	return nil
}
