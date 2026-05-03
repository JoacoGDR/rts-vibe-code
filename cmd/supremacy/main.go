// Package main is the single-binary entrypoint. The mode flag (or first
// positional argument) selects which subsystem to start:
//
//	supremacy core-api
//	supremacy gateway
//	supremacy engine
//	supremacy worker
//	supremacy ai-bot
//
// Same code, same dependency graph, different process per mode in production.
// Splitting into multiple binaries later is a one-line build change.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joaquing/clone-supremacy/internal/aibot"
	"github.com/joaquing/clone-supremacy/internal/app"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/engine"
	"github.com/joaquing/clone-supremacy/internal/gateway"
	"github.com/joaquing/clone-supremacy/internal/platform/log"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/worker"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	mode := flag.String("mode", "", "subsystem mode: core-api | gateway | engine | worker | ai-bot (also accepted as first positional arg)")
	flag.Parse()

	if *mode == "" && flag.NArg() > 0 {
		*mode = flag.Arg(0)
	}
	if *mode == "" {
		*mode = os.Getenv("SUPREMACY_MODE")
	}
	if *mode == "" {
		return errors.New("mode is required (set --mode or SUPREMACY_MODE)")
	}

	cfg, err := config.Load(*mode)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := log.New(cfg.LogLevel, cfg.LogFormat, cfg.Mode, cfg.Env)
	mreg := metrics.New(cfg.Mode)

	logger.Info("startup",
		"version", Version,
		"http_addr", cfg.HTTPAddr,
		"metrics_addr", cfg.MetricsAddr,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 2)
	go func() { errCh <- mreg.Serve(ctx, cfg.MetricsAddr) }()

	switch cfg.Mode {
	case config.ModeCoreAPI:
		go func() { errCh <- app.RunCoreAPI(ctx, cfg, logger, mreg, Version) }()
	case config.ModeGateway:
		go func() { errCh <- gateway.Run(ctx, cfg, logger, mreg, Version) }()
	case config.ModeEngine:
		go func() { errCh <- engine.Run(ctx, cfg, logger, mreg, Version) }()
	case config.ModeWorker:
		go func() { errCh <- worker.Run(ctx, cfg, logger, mreg, Version) }()
	case config.ModeAIBot:
		go func() { errCh <- aibot.Run(ctx, cfg, logger, mreg, Version) }()
	default:
		return fmt.Errorf("unknown mode %q", cfg.Mode)
	}

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
