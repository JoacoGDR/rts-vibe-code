// Package main is the engine service entrypoint (match simulation).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/engine"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), config.ModeEngine, Version, engine.Run); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
