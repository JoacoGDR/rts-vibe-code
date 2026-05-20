package natsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

// EndMatchHandler is invoked when the worker asks this engine to tear
// down an in-memory match immediately (abandonment path).
type EndMatchHandler func(payload EndMatchPayload) error

// SubscribeEndMatches listens on match.*.end for forced teardown
// requests. The handler should call [enginesvc.Runner.ForceEnd].
func SubscribeEndMatches(_ context.Context, logger *slog.Logger, nc *nats.Conn, handler EndMatchHandler) error {
	_, err := nc.Subscribe("match.*.end", func(m *nats.Msg) {
		var p EndMatchPayload
		if err := json.Unmarshal(m.Data, &p); err != nil {
			logger.Warn("bad end-match payload", "err", err)
			return
		}
		if err := handler(p); err != nil {
			logger.Warn("end-match handler failed", "err", err, "match", p.MatchID)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe end matches: %w", err)
	}
	return nil
}
