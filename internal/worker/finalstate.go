package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// FinalStateRepo is the persistence surface the final-state worker needs.
type FinalStateRepo interface {
	SaveSnapshot(ctx context.Context, matchID uuid.UUID, seq uint64, payload any) error
	MarkEnded(ctx context.Context, id uuid.UUID, winner *uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (lobbyMatch, error)
	ListPlayers(ctx context.Context, matchID uuid.UUID) ([]lobbyPlayer, error)
}

// lobbyMatch / lobbyPlayer are minimal read models so tests can stub
// without importing lobbydom.
type lobbyMatch struct {
	ID        uuid.UUID
	StartedAt *time.Time
	Status    string
}

type lobbyPlayer struct {
	UserID uuid.UUID
	Slot   string
	Alive  bool
}

// FinalStateWorker subscribes to match.*.state.final, persists the
// snapshot, marks the lobby row ended, and projects match_stats.
type FinalStateWorker struct {
	Logger *slog.Logger
	NC     *nats.Conn
	Repo   FinalStateRepo
	Stats  *pgrepo.StatsRepo
}

func (w *FinalStateWorker) Run(ctx context.Context) {
	if err := natsbridge.SubscribeFinalStates(ctx, w.Logger, w.NC, w.handle); err != nil {
		w.Logger.Error("final-state subscribe failed", "err", err)
	}
	<-ctx.Done()
}

func (w *FinalStateWorker) handle(env wire.ServerEnvelope) error {
	if env.State == nil {
		return nil
	}
	matchID, err := uuid.Parse(env.MatchID)
	if err != nil {
		return err
	}
	reqCtx := context.Background()
	if err := w.Repo.SaveSnapshot(reqCtx, matchID, env.Seq, env); err != nil {
		w.Logger.Warn("final snapshot save failed", "match", matchID, "err", err)
	}
	var winnerUID *uuid.UUID
	if uid := winnerUserID(reqCtx, w.Repo, matchID, env.State); uid != uuid.Nil {
		winnerUID = &uid
	}
	if err := w.Repo.MarkEnded(reqCtx, matchID, winnerUID); err != nil {
		w.Logger.Warn("mark ended failed", "match", matchID, "err", err)
	}
	if w.Stats != nil {
		m, _ := w.Repo.Get(reqCtx, matchID)
		row := projectStats(matchID, m.StartedAt, env.State)
		if err := w.Stats.Upsert(reqCtx, row); err != nil {
			w.Logger.Warn("stats upsert failed", "match", matchID, "err", err)
		}
	}
	w.Logger.Info("match finalized", "match", matchID, "seq", env.Seq)
	return nil
}

func winnerUserID(ctx context.Context, repo FinalStateRepo, matchID uuid.UUID, st *wire.MatchState) uuid.UUID {
	players, err := repo.ListPlayers(ctx, matchID)
	if err != nil {
		return uuid.Nil
	}
	alive := map[string]bool{}
	for _, p := range st.Players {
		if p.Alive {
			alive[p.ID] = true
		}
	}
	if len(alive) != 1 {
		return uuid.Nil
	}
	var sole string
	for s := range alive {
		sole = s
	}
	for _, p := range players {
		if p.Slot == sole {
			return p.UserID
		}
	}
	return uuid.Nil
}

// pgrepoMatchesAdapter implements FinalStateRepo atop pgrepo.Matches.
type pgrepoMatchesAdapter struct {
	*pgrepo.Matches
}

func (a pgrepoMatchesAdapter) Get(ctx context.Context, id uuid.UUID) (lobbyMatch, error) {
	m, err := a.Matches.Get(ctx, id)
	if err != nil {
		return lobbyMatch{}, err
	}
	return lobbyMatch{ID: m.ID, StartedAt: m.StartedAt, Status: string(m.Status)}, nil
}

func (a pgrepoMatchesAdapter) ListPlayers(ctx context.Context, matchID uuid.UUID) ([]lobbyPlayer, error) {
	pls, err := a.Matches.ListPlayers(ctx, matchID)
	if err != nil {
		return nil, err
	}
	out := make([]lobbyPlayer, len(pls))
	for i, p := range pls {
		out[i] = lobbyPlayer{UserID: p.UserID, Slot: p.Slot, Alive: p.Alive}
	}
	return out, nil
}

// projectStats builds a coarse post-game summary from the final snapshot.
func projectStats(matchID uuid.UUID, startedAt *time.Time, st *wire.MatchState) pgrepo.MatchStatsRow {
	perSlot := map[string]any{}
	for _, p := range st.Players {
		perSlot[p.ID] = map[string]any{"alive": p.Alive}
	}
	unitsBySlot := map[string]int{}
	for _, u := range st.Units {
		unitsBySlot[u.OwnerID]++
		if _, ok := perSlot[u.OwnerID]; ok {
			perSlot[u.OwnerID] = map[string]any{
				"alive": true, "units": unitsBySlot[u.OwnerID],
			}
		}
	}
	for slot, n := range unitsBySlot {
		entry, _ := perSlot[slot].(map[string]any)
		if entry == nil {
			entry = map[string]any{}
		}
		entry["units"] = n
		perSlot[slot] = entry
	}
	duration := 0
	if startedAt != nil && !st.GameTime.IsZero() {
		duration = int(st.GameTime.Sub(*startedAt).Seconds())
		if duration < 0 {
			duration = 0
		}
	}
	capitals := 0
	for _, p := range st.Provinces {
		if p.Capital && p.OwnerID != "" {
			capitals++
		}
	}
	return pgrepo.MatchStatsRow{
		MatchID: matchID, DurationSec: duration,
		TotalUnits: len(st.Units), TotalCombats: 0,
		CapitalsTaken: capitals, PerSlot: perSlot,
	}
}
