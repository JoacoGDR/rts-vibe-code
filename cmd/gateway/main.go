// Package main is the gateway service entrypoint (WebSocket edge).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/gateway"
	"github.com/joaquing/clone-supremacy/internal/platform/bootstrap"
)

// Version is overwritten via -ldflags at build time.
var Version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), config.ModeGateway, Version, gateway.Run); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
