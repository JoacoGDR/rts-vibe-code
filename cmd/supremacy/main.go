// Package main is the multi-mode entrypoint for local dev and integration
// tests. Production deploys use the dedicated binaries under cmd/<service>/.
//
//	supremacy core-api
//	supremacy gateway
//	supremacy engine
//	supremacy worker
//	supremacy ai-bot
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/aibot"
	"github.com/joaquing/clone-supremacy/internal/app"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/engine"
	"github.com/joaquing/clone-supremacy/internal/gateway"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
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

	start, err := runnerFor(*mode)
	if err != nil {
		return err
	}
	return bootstrap.Run(context.Background(), *mode, Version, start)
}

func runnerFor(mode string) (bootstrap.Runner, error) {
	switch mode {
	case config.ModeCoreAPI:
		return app.RunCoreAPI, nil
	case config.ModeGateway:
		return gateway.Run, nil
	case config.ModeEngine:
		return engine.Run, nil
	case config.ModeWorker:
		return worker.Run, nil
	case config.ModeAIBot:
		return aibot.Run, nil
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
}
