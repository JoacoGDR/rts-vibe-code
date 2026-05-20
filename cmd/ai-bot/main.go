// Package main is the ai-bot service entrypoint (headless player clients).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/aibot"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), config.ModeAIBot, Version, aibot.Run); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
