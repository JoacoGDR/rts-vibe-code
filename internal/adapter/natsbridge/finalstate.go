package natsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// FinalStateHandler receives the public final snapshot envelope.
type FinalStateHandler func(env wire.ServerEnvelope) error

// SubscribeFinalStates listens on match.*.state.final for the one-shot
// post-game snapshot the engine publishes after cleanup.
func SubscribeFinalStates(_ context.Context, logger *slog.Logger, nc *nats.Conn, handler FinalStateHandler) error {
	_, err := nc.Subscribe("match.*.state.final", func(m *nats.Msg) {
		var env wire.ServerEnvelope
		if err := json.Unmarshal(m.Data, &env); err != nil {
			logger.Warn("bad final-state payload", "err", err)
			return
		}
		if err := handler(env); err != nil {
			logger.Warn("final-state handler failed", "err", err, "match", env.MatchID)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe final states: %w", err)
	}
	return nil
}
