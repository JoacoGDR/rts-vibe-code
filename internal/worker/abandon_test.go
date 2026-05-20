package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

type abandonFakeRepo struct {
	active    []lobbydom.Match
	abandoned uuid.UUID
}

func (r *abandonFakeRepo) ActiveMatches(context.Context) ([]lobbydom.Match, error) {
	return r.active, nil
}

func (r *abandonFakeRepo) MatchAbandonedCandidate(_ context.Context, matchID uuid.UUID, _ int64) (bool, error) {
	return matchID == r.abandoned, nil
}

func (r *abandonFakeRepo) MarkAbandoned(_ context.Context, id uuid.UUID) error {
	r.abandoned = id
	return nil
}

func TestAbandonScanMarksMatch(t *testing.T) {
	id := uuid.New()
	repo := &abandonFakeRepo{
		active:    []lobbydom.Match{{ID: id, Status: lobbydom.StatusActive}},
		abandoned: id,
	}
	w := &AbandonWorker{
		Repo: repo, Threshold: time.Minute, ScanEvery: time.Hour,
	}
	w.scan(context.Background())
	if repo.abandoned != id {
		t.Fatalf("expected abandoned %s", id)
	}
}
