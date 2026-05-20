package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/log"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
)

// Runner is the mode-specific composition root each binary links.
type Runner func(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error

// Run loads configuration for mode, starts metrics, invokes start, and blocks
// until shutdown or a fatal error from either goroutine.
func Run(ctx context.Context, mode, version string, start Runner) error {
	cfg, err := config.Load(mode)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := log.New(cfg.LogLevel, cfg.LogFormat, cfg.Mode, cfg.Env)
	mreg := metrics.New(cfg.Mode)

	logger.Info("startup",
		"version", version,
		"http_addr", cfg.HTTPAddr,
		"metrics_addr", cfg.MetricsAddr,
	)

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 2)
	go func() { errCh <- mreg.Serve(ctx, cfg.MetricsAddr) }()
	go func() { errCh <- start(ctx, cfg, logger, mreg, version) }()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}
