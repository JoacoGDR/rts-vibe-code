package enginesvc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/cmddom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/visibility"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// ErrNotHosted is returned by Submit when the match is not currently
// running on this engine instance. Composition-root wiring translates this
// into the transport-level NAK signal.
var ErrNotHosted = errors.New("match not hosted on this engine")

// Runner owns the lifecycle of every match this engine instance is hosting.
// One match = one goroutine driving its own mini event loop.
type Runner struct {
	logger      *slog.Logger
	metrics     *metrics.Registry
	broadcaster Broadcaster
	slots       SlotIndex
	mu          sync.Mutex
	matches     map[string]*matchHandle
	tickEvery   time.Duration
}

type matchHandle struct {
	match  *matchdom.Match
	cmds   chan cmddom.Command
	cancel context.CancelFunc
}

// NewRunner constructs a Runner with the given dependencies. A nil
// SlotIndex is replaced with a no-op so tests don't have to wire one.
func NewRunner(logger *slog.Logger, mreg *metrics.Registry, broadcaster Broadcaster, slots SlotIndex) *Runner {
	if slots == nil {
		slots = noopSlotIndex{}
	}
	return &Runner{
		logger:      logger,
		metrics:     mreg,
		broadcaster: broadcaster,
		slots:       slots,
		matches:     map[string]*matchHandle{},
		tickEvery:   250 * time.Millisecond,
	}
}

// EnsureMatch starts a match goroutine if one is not already running for
// the given match id. Idempotent.
func (r *Runner) EnsureMatch(parent context.Context, m *matchdom.Match) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.matches[m.ID]; ok {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	h := &matchHandle{
		match:  m,
		cmds:   make(chan cmddom.Command, 64),
		cancel: cancel,
	}
	r.matches[m.ID] = h
	go r.runMatch(ctx, h)
}

// Submit forwards a command to its match. Returns ErrNotHosted if the
// engine is not hosting that match locally; the consumer should NAK so
// another engine can try.
func (r *Runner) Submit(cmd cmddom.Command) error {
	r.mu.Lock()
	h, ok := r.matches[string(cmd.MatchID)]
	r.mu.Unlock()
	if !ok {
		return ErrNotHosted
	}
	select {
	case h.cmds <- cmd:
		return nil
	default:
		return errors.New("match command queue full")
	}
}

// IsHosted reports whether the given match is currently being driven by
// this runner.
func (r *Runner) IsHosted(matchID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.matches[matchID]
	return ok
}

// Rebroadcast publishes the latest filtered state for every slot, used to
// answer client `resync` messages.
func (r *Runner) Rebroadcast(ctx context.Context, matchID string) {
	r.mu.Lock()
	h, ok := r.matches[matchID]
	r.mu.Unlock()
	if !ok {
		return
	}
	r.advanceTime(h.match)
	r.broadcastState(ctx, h.match)
}

// BroadcastInitial publishes the post-start snapshot for a brand new
// match. Called once by the start subscriber.
func (r *Runner) BroadcastInitial(ctx context.Context, m *matchdom.Match) {
	r.broadcastState(ctx, m)
}

// SlotsIndex exposes the configured slot index for adapters that need to
// seed it at match start.
func (r *Runner) SlotsIndex() SlotIndex { return r.slots }

// Stop tears down a hosted match.
func (r *Runner) Stop(matchID string) {
	r.mu.Lock()
	h, ok := r.matches[matchID]
	if ok {
		delete(r.matches, matchID)
	}
	r.mu.Unlock()
	if ok {
		h.cancel()
	}
}

func (r *Runner) runMatch(ctx context.Context, h *matchHandle) {
	logger := r.logger.With("match_id", h.match.ID)
	logger.Info("match loop starting", "map", h.match.MapID, "speed", h.match.Speed)
	ticker := time.NewTicker(r.tickEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("match loop stopping")
			return
		case cmd := <-h.cmds:
			r.advanceTime(h.match)
			result, immediate, err := cmddom.Dispatch(h.match, cmd)
			if err != nil {
				r.metrics.CommandsApplied.WithLabelValues(cmd.Kind, result).Inc()
				logger.Warn("command rejected", "kind", cmd.Kind, "err", err)
				continue
			}
			r.metrics.CommandsApplied.WithLabelValues(cmd.Kind, result).Inc()
			logger.Debug("command applied", "kind", cmd.Kind, "unit", cmd.UnitID)
			events := make([]matchdom.AppliedEvent, 0, len(immediate))
			events = append(events, immediate...)
			events = append(events, h.match.ProcessNext(h.match.GameNow)...)
			r.broadcastEvents(ctx, h.match, events)
		case <-ticker.C:
			r.advanceTime(h.match)
			events := h.match.ProcessNext(h.match.GameNow)
			if len(events) > 0 {
				r.broadcastEvents(ctx, h.match, events)
			}
		}
		if h.match.Status == "ended" {
			logger.Info("match ended", "winner", h.match.WinnerSlot)
			r.broadcastState(ctx, h.match)
			r.Stop(h.match.ID)
			return
		}
	}
}

// advanceTime ticks the in-memory game clock based on real-wall elapsed
// time.
func (r *Runner) advanceTime(m *matchdom.Match) {
	now := time.Now()
	wallElapsed := now.Sub(m.StartedAt)
	m.GameNow = m.GameStart.Add(time.Duration(float64(wallElapsed) * m.Speed))
}

func (r *Runner) broadcastEvents(ctx context.Context, m *matchdom.Match, events []matchdom.AppliedEvent) {
	if r.broadcaster == nil {
		return
	}
	for _, e := range events {
		r.metrics.EngineEvents.WithLabelValues(e.Kind).Inc()
		r.publishEventPerSlot(ctx, m, e)
	}
	r.broadcastState(ctx, m)
}

func (r *Runner) publishEventPerSlot(ctx context.Context, m *matchdom.Match, e matchdom.AppliedEvent) {
	env := wire.ServerEnvelope{
		Type:    wire.ServerEvent,
		MatchID: m.ID,
		Seq:     e.Seq,
		SentAt:  time.Now(),
		Event: &wire.Event{
			Kind: e.Kind, OccurAt: e.OccurAt,
			UnitID: e.UnitID, Province: e.Province, Extra: e.Extra,
		},
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return
	}

	_ = r.broadcaster.PublishPublicEvent(ctx, m.ID, payload)
	for slot := range m.Players {
		if !slotObservesEvent(m, slot, e) {
			continue
		}
		_ = r.broadcaster.PublishSlotEvent(ctx, m.ID, slot, payload)
	}
}

// slotObservesEvent reports whether a slot should be told about an event,
// given the visibility computed for this match at the moment the event
// fires.
func slotObservesEvent(m *matchdom.Match, slot string, e matchdom.AppliedEvent) bool {
	provinces, units := visibility.For(m, slot, m.GameNow)
	if e.Slot == slot {
		return true
	}
	if e.UnitID != "" && units[e.UnitID] {
		return true
	}
	if e.Province != "" && provinces[e.Province] {
		return true
	}
	switch e.Kind {
	case "match_ended", "province_captured":
		return true
	}
	return false
}

func (r *Runner) broadcastState(ctx context.Context, m *matchdom.Match) {
	if r.broadcaster == nil {
		return
	}
	publicEnv := wire.ServerEnvelope{
		Type: wire.ServerState, MatchID: m.ID,
		Seq: m.Seq, SentAt: time.Now(), State: visibility.Snapshot(m),
	}
	if payload, err := json.Marshal(publicEnv); err == nil {
		_ = r.broadcaster.PublishPublicState(ctx, m.ID, payload)
	}

	for slot := range m.Players {
		env := wire.ServerEnvelope{
			Type: wire.ServerState, MatchID: m.ID,
			Seq: m.Seq, SentAt: time.Now(), State: visibility.Filtered(m, slot),
		}
		payload, err := json.Marshal(env)
		if err != nil {
			continue
		}
		_ = r.broadcaster.PublishSlotState(ctx, m.ID, slot, payload)
	}
}
