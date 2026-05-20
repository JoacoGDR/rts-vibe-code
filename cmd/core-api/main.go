// Package main is the core-api service entrypoint (REST, chat persistence).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/app"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), config.ModeCoreAPI, Version, app.RunCoreAPI); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
