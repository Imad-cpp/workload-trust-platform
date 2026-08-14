package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const defaultListenAddr = "127.0.0.1:8080"

type Config struct {
	ListenAddr  string
	DatabaseURL string
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

	cfg := Config{
		ListenAddr:  listenAddr,
		DatabaseURL: databaseURL,
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.ListenAddr)
	if err != nil {
		return fmt.Errorf("invalid WTP_LISTEN_ADDR: %w", err)
	}
	if port == "" {
		return errors.New("WTP_LISTEN_ADDR must include a port")
	}

	if host == "localhost" {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("WTP_LISTEN_ADDR must bind to a loopback address during the unauthenticated Phase 2 API")
	}
	return nil
}
