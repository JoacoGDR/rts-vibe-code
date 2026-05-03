package aibot

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/health"
	"github.com/joaquing/clone-supremacy/internal/platform/httpserver"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/storage"
)

// Run is the composition root for the `ai-bot` binary mode. It wires
// Postgres + a core-api HTTP client onto a [Runner] and serves the
// health probe alongside.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error {
	pool, err := storage.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := storage.Migrate(ctx, pool); err != nil {
		return err
	}

	core := NewCoreClient(cfg.AIBotCoreAPIURL, cfg.BotAPIKey)
	runner := NewRunner(logger, mreg, pgrepo.NewMatches(pool), core, cfg.AIBotGatewayURL, cfg.AIPollInterval)

	go func() {
		if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("ai-bot runner stopped", "err", err)
		}
	}()

	h := health.New(cfg.Mode, version)
	h.RegisterReady(func() error {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			return err
		}
		if runner.PoolSize() == 0 {
			return errors.New("bot pool is empty")
		}
		return nil
	})

	r := chi.NewRouter()
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)

	return httpserver.Run(ctx, httpserver.Config{
		Addr:    cfg.HTTPAddr,
		Handler: r,
		Logger:  logger,
		Name:    "ai-bot",
	})
}
