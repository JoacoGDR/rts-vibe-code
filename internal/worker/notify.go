package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/internal/service/notifysvc"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// NotifyWorker is the second consumer the worker mode runs. It
// subscribes to the public match event stream, picks events that map
// onto a per-user notification (via [eventToKind]), looks up the
// affected user(s) and writes through to [notifysvc] which handles
// debouncing + persistence. A separate dispatch loop drains the
// pending-delivery queue and writes to SMTP (or fakes the send when
// SMTP is not configured).
type NotifyWorker struct {
	Logger    *slog.Logger
	NC        *nats.Conn
	Notify    *notifysvc.Service
	Players   PlayerLookup
	Mailer    Mailer
	BatchSize int
	Poll      time.Duration

	cache    sync.Map // matchID -> []lobbydom.Player
	cacheTTL time.Duration
	cacheAt  sync.Map // matchID -> time.Time
}

// PlayerLookup is the slice of pgrepo.Matches the notify worker uses.
// Pulled out for tests and to keep the dependency arrow pointing
// outwards.
type PlayerLookup interface {
	ListPlayers(ctx context.Context, matchID uuid.UUID) ([]playerLookupResult, error)
}

// playerLookupResult is what the notify worker actually consumes from
// the players list — slot, user id, alive flag. We deliberately keep
// it scoped to this package so the worker doesn't drag the lobbydom
// type in its public surface.
type playerLookupResult struct {
	UserID         uuid.UUID
	Slot           string
	Alive          bool
	ControlledByAI bool
}

// matchPlayerLookup adapts *pgrepo.Matches onto PlayerLookup.
type matchPlayerLookup struct{ Repo *pgrepo.Matches }

func (l matchPlayerLookup) ListPlayers(ctx context.Context, matchID uuid.UUID) ([]playerLookupResult, error) {
	players, err := l.Repo.ListPlayers(ctx, matchID)
	if err != nil {
		return nil, err
	}
	out := make([]playerLookupResult, 0, len(players))
	for _, p := range players {
		out = append(out, playerLookupResult{
			UserID: p.UserID, Slot: p.Slot, Alive: p.Alive, ControlledByAI: p.ControlledByAI,
		})
	}
	return out, nil
}

// Run wires the event subscription and the SMTP dispatch loop. Returns
// when ctx is cancelled.
func (w *NotifyWorker) Run(ctx context.Context) {
	if w.BatchSize <= 0 {
		w.BatchSize = 50
	}
	if w.Poll <= 0 {
		w.Poll = 15 * time.Second
	}
	w.cacheTTL = 30 * time.Second

	if _, err := w.NC.Subscribe(natsbridge.SubjEventAll, func(m *nats.Msg) {
		w.handleEvent(ctx, m.Data)
	}); err != nil {
		w.Logger.Error("notify event subscribe failed", "err", err)
		return
	}

	t := time.NewTicker(w.Poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.dispatch(ctx)
		}
	}
}

// handleEvent decodes one server envelope and fans the event out into
// notifications for every user it targets. Best-effort — we never NAK
// since the worker is a side-channel, not a transactional outbox.
func (w *NotifyWorker) handleEvent(ctx context.Context, raw []byte) {
	var env wire.ServerEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return
	}
	if env.Event == nil || env.MatchID == "" {
		return
	}
	matchID, err := uuid.Parse(env.MatchID)
	if err != nil {
		return
	}
	templates := buildTemplates(env.Event)
	if len(templates) == 0 {
		return
	}
	players, err := w.players(ctx, matchID)
	if err != nil {
		w.Logger.Warn("notify: players lookup failed", "err", err, "match", matchID)
		return
	}
	for _, tpl := range templates {
		for _, target := range tpl.Targets(env.Event, players) {
			if _, err := w.Notify.Insert(ctx, notifysvc.Insert{
				UserID: target, MatchID: matchID, Kind: tpl.Kind, Payload: tpl.Payload,
			}); err != nil {
				w.Logger.Warn("notify insert failed", "err", err, "kind", tpl.Kind)
			}
		}
	}
}

// dispatch pulls the next batch of unsent notifications and tries to
// deliver them. Mailer failures are logged but do not block the loop —
// we'll retry next tick.
func (w *NotifyWorker) dispatch(ctx context.Context) {
	pending, err := w.Notify.PendingDelivery(ctx, w.BatchSize)
	if err != nil {
		w.Logger.Warn("notify pending fetch failed", "err", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	for _, n := range pending {
		w.dispatchOne(ctx, n)
	}
}

func (w *NotifyWorker) dispatchOne(ctx context.Context, n notifydom.Notification) {
	if w.Mailer == nil {
		w.Logger.Debug("notify mailer unset, marking sent", "id", n.ID, "kind", n.Kind)
		_ = w.Notify.MarkSent(ctx, n.ID)
		return
	}
	if err := w.Mailer.Send(ctx, n); err != nil {
		w.Logger.Warn("notify mailer send failed", "err", err, "id", n.ID)
		return
	}
	if err := w.Notify.MarkSent(ctx, n.ID); err != nil {
		w.Logger.Warn("notify mark-sent failed", "err", err, "id", n.ID)
	}
}

// players returns the cached match roster, refreshing if the cache is
// stale. Caching keeps the worker from hammering Postgres when a
// combat sequence emits a burst of events.
func (w *NotifyWorker) players(ctx context.Context, matchID uuid.UUID) ([]playerLookupResult, error) {
	if v, ok := w.cache.Load(matchID); ok {
		if at, ok := w.cacheAt.Load(matchID); ok {
			if t, ok := at.(time.Time); ok && time.Since(t) < w.cacheTTL {
				if list, ok := v.([]playerLookupResult); ok {
					return list, nil
				}
			}
		}
	}
	if w.Players == nil {
		return nil, errors.New("notify worker: players lookup not configured")
	}
	list, err := w.Players.ListPlayers(ctx, matchID)
	if err != nil {
		return nil, err
	}
	w.cache.Store(matchID, list)
	w.cacheAt.Store(matchID, time.Now())
	return list, nil
}
