package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// PresenceRepo is the slice of pgrepo.Matches the presence worker
// depends on. Pulled out as an interface so unit tests can stub it.
type PresenceRepo interface {
	TouchPresence(ctx context.Context, matchID uuid.UUID, slot string, at time.Time) error
	SlotsNeedingTakeover(ctx context.Context, threshold time.Duration) ([]pgrepo.InactiveSlot, error)
	MarkControlledByAI(ctx context.Context, matchID uuid.UUID, slot string, on bool) (bool, error)
}

// PresenceWorker subscribes to the gateway's per-(match, slot)
// presence pings (one per WebSocket session) and records them as
// match_players.last_seen_at. A periodic scanner — configured to run
// every hour by default — flips any slot whose last_seen_at falls
// behind the configured threshold (default 72h) to controlled_by_ai.
type PresenceWorker struct {
	Logger        *slog.Logger
	NC            *nats.Conn
	Repo          PresenceRepo
	TakeoverAfter time.Duration
	ScanEvery     time.Duration
	// OnAITakeover lets in-process callers (e.g. the notification
	// worker, when stitched into the same binary) react to a takeover
	// without a second DB read. May be nil.
	OnAITakeover func(ctx context.Context, slot pgrepo.InactiveSlot)
}

// Run wires the NATS subscription and the periodic scanner. Returns
// when ctx is cancelled.
func (w *PresenceWorker) Run(ctx context.Context) {
	if w.TakeoverAfter <= 0 {
		w.TakeoverAfter = 72 * time.Hour
	}
	if w.ScanEvery <= 0 {
		w.ScanEvery = time.Hour
	}

	if err := natsbridge.SubscribePresence(ctx, w.Logger, w.NC, w.handlePresence); err != nil {
		w.Logger.Error("presence subscribe failed", "err", err)
		return
	}

	t := time.NewTicker(w.ScanEvery)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.scan(ctx)
		}
	}
}

// handlePresence persists the wall-clock time the worker observed for
// the given (match, slot). Errors are logged but never re-thrown — a
// failed update only delays AI takeover.
func (w *PresenceWorker) handlePresence(ctx context.Context, p natsbridge.PresencePayload) error {
	matchID, err := uuid.Parse(p.MatchID)
	if err != nil {
		return err
	}
	at := p.At
	if at.IsZero() {
		at = time.Now()
	}
	return w.Repo.TouchPresence(ctx, matchID, p.Slot, at)
}

// scan walks the slots that have been silent past the threshold and
// flips them to AI control. The OnAITakeover hook lets callers (e.g.
// the notification worker) react in-process without another DB read.
func (w *PresenceWorker) scan(ctx context.Context) {
	slots, err := w.Repo.SlotsNeedingTakeover(ctx, w.TakeoverAfter)
	if err != nil {
		w.Logger.Warn("ai-takeover scan failed", "err", err)
		return
	}
	for _, s := range slots {
		changed, err := w.Repo.MarkControlledByAI(ctx, s.MatchID, s.Slot, true)
		if err != nil {
			w.Logger.Warn("ai-takeover flip failed", "err", err, "match", s.MatchID, "slot", s.Slot)
			continue
		}
		if !changed {
			continue
		}
		w.Logger.Info("ai-takeover triggered", "match", s.MatchID, "slot", s.Slot)
		w.publishTakeoverEvent(s)
		if w.OnAITakeover != nil {
			w.OnAITakeover(ctx, s)
		}
	}
}

// publishTakeoverEvent fans out a `bot_takeover` event on the public
// match event subject so observers (frontend banner, notification
// worker) can react. Best-effort — the canonical truth is the DB flag.
func (w *PresenceWorker) publishTakeoverEvent(s pgrepo.InactiveSlot) {
	if w.NC == nil {
		return
	}
	env := wire.ServerEnvelope{
		Type:    wire.ServerEvent,
		MatchID: s.MatchID.String(),
		SentAt:  time.Now(),
		Event: &wire.Event{
			Kind:    "bot_takeover",
			OccurAt: time.Now(),
			Extra: map[string]any{
				"slot":         s.Slot,
				"prev_user_id": s.UserID.String(),
			},
		},
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return
	}
	_ = w.NC.Publish(natsbridge.PublicEventSubject(s.MatchID.String()), payload)
}
