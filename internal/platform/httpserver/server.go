package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Config tunes the HTTP server. Zero values get sensible defaults.
type Config struct {
	Addr            string
	Handler         http.Handler
	ReadHeader      time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
	Name            string // shows up in startup/shutdown logs
}

// Run starts the HTTP listener and blocks until either ctx is cancelled
// (graceful shutdown) or the underlying server errors out (non-graceful).
// On graceful shutdown a fresh context bounded by Config.ShutdownTimeout
// drives Server.Shutdown so in-flight requests get a chance to finish.
func Run(ctx context.Context, cfg Config) error {
	if cfg.ReadHeader == 0 {
		cfg.ReadHeader = 5 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 60 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           cfg.Handler,
		ReadHeaderTimeout: cfg.ReadHeader,
		IdleTimeout:       cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	if cfg.Logger != nil {
		cfg.Logger.Info("http listening", "name", cfg.Name, "addr", cfg.Addr)
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		if cfg.Logger != nil {
			cfg.Logger.Info("http shutdown", "name", cfg.Name, "err", err)
		}
		return err
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
