package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/audit"
	"github.com/Imad-cpp/workload-trust-platform/internal/config"
	"github.com/Imad-cpp/workload-trust-platform/internal/database"
	"github.com/Imad-cpp/workload-trust-platform/internal/reconcile"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/spiremgmt"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("SPIRE reconciliation failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.LoadReconciler()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	spire, err := spiremgmt.NewUnixClient(cfg.SPIREServerSocket)
	if err != nil {
		return err
	}
	defer func() { _ = spire.Close() }()

	service, err := reconcile.New(
		registration.NewPostgresRepository(pool),
		spire,
		audit.NewPostgresRepository(pool),
	)
	if err != nil {
		return err
	}
	summary, err := service.Run(ctx)
	logger.Info(
		"SPIRE reconciliation completed",
		"total", summary.Total,
		"converged", summary.Converged,
		"changed", summary.Changed,
		"failed", summary.Failed,
	)
	return err
}
