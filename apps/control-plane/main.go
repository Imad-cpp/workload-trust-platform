package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/config"
	"github.com/Imad-cpp/workload-trust-platform/internal/database"
	"github.com/Imad-cpp/workload-trust-platform/internal/httpapi"
	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
	"github.com/Imad-cpp/workload-trust-platform/internal/policy"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	authenticator, err := operatorauth.NewStaticBearer(cfg.OperatorToken, cfg.OperatorID, cfg.OperatorRole)
	if err != nil {
		return err
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(rootCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	workloads := workload.NewPostgresRepository(pool)
	api, err := httpapi.New(httpapi.Dependencies{
		Readiness:             pool,
		Workloads:             workloads,
		RegistrationMutations: registration.NewPostgresMutationService(pool),
		PolicyManager:         policy.NewPostgresManager(pool),
		Authenticator:         authenticator,
		Authorizer:            operatorauth.RBAC{},
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("control plane listening", "address", cfg.ListenAddr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-rootCtx.Done():
		logger.Info("control plane shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-serveErr
}
