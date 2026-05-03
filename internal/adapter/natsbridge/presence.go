package natsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// SubjPresenceAll is the wildcard the worker subscribes on. The
// gateway publishes per-(match, slot) under match.<id>.presence.<slot>
// once per WebSocket session — no periodic heartbeat. Core NATS, not
// JetStream, because losing a single ping only delays AI takeover by
// at most one scan tick, and the threshold is measured in days.
const SubjPresenceAll = "match.*.presence.*"

// PresenceSubject builds the per-(match, slot) presence subject.
func PresenceSubject(matchID, slot string) string {
	return fmt.Sprintf("match.%s.presence.%s", matchID, slot)
}

// PresencePayload is the on-wire shape the gateway publishes when a
// user opens a match WebSocket. We stamp the wall-clock instant the
// gateway observed the connection so the worker doesn't have to trust
// client clocks.
type PresencePayload struct {
	MatchID string    `json:"match_id"`
	UserID  string    `json:"user_id"`
	Slot    string    `json:"slot"`
	At      time.Time `json:"at"`
}

// PublishPresence is the gateway-side publisher. Best-effort: send
// failures are swallowed because presence is advisory.
func (p *Publisher) PublishPresence(_ context.Context, payload PresencePayload) error {
	if payload.MatchID == "" || payload.Slot == "" {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.NC.Publish(PresenceSubject(payload.MatchID, payload.Slot), raw)
}

// PresenceHandler is invoked once per delivered presence ping after
// the payload has been decoded.
type PresenceHandler func(ctx context.Context, payload PresencePayload) error

// SubscribePresence wires a core NATS subscription onto the wildcard.
// Bad payloads are logged and dropped.
func SubscribePresence(ctx context.Context, logger *slog.Logger, nc *nats.Conn, handler PresenceHandler) error {
	_, err := nc.Subscribe(SubjPresenceAll, func(m *nats.Msg) {
		var p PresencePayload
		if err := json.Unmarshal(m.Data, &p); err != nil {
			logger.Warn("bad presence payload", "err", err)
			return
		}
		if err := handler(ctx, p); err != nil {
			logger.Warn("presence handler failed", "err", err, "match", p.MatchID)
		}
	})
	return err
}
