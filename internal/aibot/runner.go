package aibot

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
)

// Runner owns the lifecycle of every active bot session. It polls
// Postgres on a fixed cadence for AI-controlled slots and starts a
// session goroutine for each one. The session goroutine exits when the
// slot is no longer AI-controlled (or the match ends, or the parent
// context is cancelled).
type Runner struct {
	logger     *slog.Logger
	metrics    *metrics.Registry
	matches    *pgrepo.Matches
	core       *CoreClient
	gateway    string
	pool       *botPool
	pollEvery  time.Duration
	maxBacklog int

	mu       sync.Mutex
	sessions map[sessionKey]context.CancelFunc
}

type sessionKey struct {
	matchID uuid.UUID
	slot    string
}

// NewRunner builds a Runner ready to serve. The pool argument can be
// nil for tests that drive the runner directly.
func NewRunner(logger *slog.Logger, mreg *metrics.Registry, matches *pgrepo.Matches,
	core *CoreClient, gateway string, pollEvery time.Duration) *Runner {
	if pollEvery <= 0 {
		pollEvery = 30 * time.Second
	}
	return &Runner{
		logger:     logger,
		metrics:    mreg,
		matches:    matches,
		core:       core,
		gateway:    gateway,
		pool:       newBotPool(nil),
		pollEvery:  pollEvery,
		maxBacklog: 32,
		sessions:   map[sessionKey]context.CancelFunc{},
	}
}

// PoolSize is the readiness probe view of how many bot identities are
// available right now.
func (r *Runner) PoolSize() int { return r.pool.size() }

// Run is the binary mode's main loop. It does an initial pool refresh
// + scan, then re-scans every pollEvery until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) error {
	r.refreshPool(ctx)
	r.scan(ctx)

	t := time.NewTicker(r.pollEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			r.shutdown()
			return ctx.Err()
		case <-t.C:
			r.refreshPool(ctx)
			r.scan(ctx)
		}
	}
}

// refreshPool re-loads the bot identity pool from core-api. Failures
// are logged but non-fatal — the runner falls back to the previous
// pool snapshot.
func (r *Runner) refreshPool(ctx context.Context) {
	bots, err := r.core.ListBots(ctx)
	if err != nil {
		r.logger.Warn("bot pool refresh failed", "err", err)
		return
	}
	r.pool.reload(bots)
}

// scan walks the AI-controlled slot table and reconciles in-memory
// sessions. Each new (match, slot) pair gets a fresh goroutine; pairs
// that are no longer AI-controlled get cancelled.
func (r *Runner) scan(ctx context.Context) {
	slots, err := r.matches.AISlots(ctx)
	if err != nil {
		r.logger.Warn("ai slots scan failed", "err", err)
		return
	}

	wanted := map[sessionKey]pgrepo.InactiveSlot{}
	for _, s := range slots {
		wanted[sessionKey{matchID: s.MatchID, slot: s.Slot}] = s
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for k, cancel := range r.sessions {
		if _, ok := wanted[k]; !ok {
			cancel()
			delete(r.sessions, k)
		}
	}

	for k, s := range wanted {
		if _, ok := r.sessions[k]; ok {
			continue
		}
		botID, err := r.pool.take()
		if err != nil {
			if !errors.Is(err, ErrNoBotsAvailable) {
				r.logger.Warn("bot pool take failed", "err", err)
			}
			continue
		}
		sessCtx, cancel := context.WithCancel(ctx)
		r.sessions[k] = cancel
		go r.runSession(sessCtx, s, botID)
	}
}

// runSession is the goroutine the runner spawns per AI-controlled slot.
// It builds a session, runs it, and logs the exit reason. Failures
// release the slot from the in-memory map so the next scan tick can
// retry.
func (r *Runner) runSession(ctx context.Context, slot pgrepo.InactiveSlot, botID uuid.UUID) {
	logger := r.logger.With("match", slot.MatchID, "slot", slot.Slot, "bot", botID)
	logger.Info("bot session starting")
	defer logger.Info("bot session stopped")

	defer func() {
		r.mu.Lock()
		delete(r.sessions, sessionKey{matchID: slot.MatchID, slot: slot.Slot})
		r.mu.Unlock()
	}()

	sess := newSession(r.logger, r.core, r.gateway, slot.MatchID.String(), slot.Slot, slot.UserID, botID)
	if err := sess.run(ctx); err != nil && ctx.Err() == nil {
		logger.Warn("bot session ended with error", "err", err)
	}
}

// shutdown cancels every active session. Called when the parent ctx
// is cancelled.
func (r *Runner) shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, cancel := range r.sessions {
		cancel()
	}
	r.sessions = map[sessionKey]context.CancelFunc{}
}
