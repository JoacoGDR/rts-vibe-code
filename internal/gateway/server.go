package gateway

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/mw"
	"github.com/joaquing/clone-supremacy/internal/auth"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/health"
	"github.com/joaquing/clone-supremacy/internal/platform/httpserver"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/storage"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error {
	rdb, err := storage.NewRedis(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	nc, js, err := storage.NewNATS(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	tickets := auth.NewTicketBroker(rdb, cfg.WSTicketTTL)
	hub := newHub(hubConfig{
		Logger:    logger,
		Metrics:   mreg,
		NC:        nc,
		JS:        js,
		RDB:       rdb,
		Tickets:   tickets,
		RateLimit: cfg.CommandRateMax,
	})

	h := health.New(cfg.Mode, version)
	h.RegisterReady(func() error {
		if !nc.IsConnected() {
			return errors.New("nats disconnected")
		}
		return nil
	})

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(mw.CORS)

	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)
	r.Get("/ws", hub.handleWS)

	runErr := make(chan error, 1)
	go func() {
		runErr <- httpserver.Run(ctx, httpserver.Config{
			Addr:    cfg.HTTPAddr,
			Handler: r,
			Logger:  logger,
			Name:    "gateway",
		})
	}()

	select {
	case <-ctx.Done():
		hub.shutdown()
		return <-runErr
	case err := <-runErr:
		hub.shutdown()
		return err
	}
}
