// Package main is the worker service entrypoint (background jobs).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
	"github.com/joaquing/clone-supremacy/internal/worker"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), config.ModeWorker, Version, worker.Run); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
