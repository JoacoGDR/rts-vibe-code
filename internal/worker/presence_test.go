package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
)

// fakeRepo is the in-memory PresenceRepo used by the worker tests. It
// captures the latest TouchPresence per (match, slot) and simulates
// `SlotsNeedingTakeover` by reading off whatever the test planted.
type fakeRepo struct {
	mu       sync.Mutex
	touched  map[string]time.Time
	pending  []pgrepo.InactiveSlot
	flipped  map[string]bool
	flipErr  error
	listErr  error
	touchErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		touched: map[string]time.Time{},
		flipped: map[string]bool{},
	}
}

func (r *fakeRepo) TouchPresence(_ context.Context, matchID uuid.UUID, slot string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.touchErr != nil {
		return r.touchErr
	}
	r.touched[matchID.String()+"/"+slot] = at
	return nil
}

func (r *fakeRepo) SlotsNeedingTakeover(_ context.Context, _ time.Duration) ([]pgrepo.InactiveSlot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := append([]pgrepo.InactiveSlot{}, r.pending...)
	return out, nil
}

func (r *fakeRepo) MarkControlledByAI(_ context.Context, matchID uuid.UUID, slot string, on bool) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flipErr != nil {
		return false, r.flipErr
	}
	key := matchID.String() + "/" + slot
	if r.flipped[key] == on {
		return false, nil
	}
	r.flipped[key] = on
	return true, nil
}

func TestPresenceHandlerStoresTimestamp(t *testing.T) {
	repo := newFakeRepo()
	w := &PresenceWorker{Logger: slog.Default(), Repo: repo, ScanEvery: time.Hour, TakeoverAfter: time.Hour}
	mid := uuid.New()
	when := time.Now().UTC()
	payload := natsbridge.PresencePayload{MatchID: mid.String(), Slot: "red", At: when}
	if err := w.handlePresence(context.Background(), payload); err != nil {
		t.Fatalf("handlePresence: %v", err)
	}
	if got := repo.touched[mid.String()+"/red"]; !got.Equal(when) {
		t.Fatalf("expected stored time %v, got %v", when, got)
	}
}

func TestScanFlipsExpiredSlot(t *testing.T) {
	repo := newFakeRepo()
	mid, uid := uuid.New(), uuid.New()
	repo.pending = []pgrepo.InactiveSlot{{MatchID: mid, UserID: uid, Slot: "blue"}}

	var fired []pgrepo.InactiveSlot
	w := &PresenceWorker{
		Logger: slog.Default(), Repo: repo,
		TakeoverAfter: 5 * time.Second,
		ScanEvery:     time.Hour,
		OnAITakeover:  func(_ context.Context, s pgrepo.InactiveSlot) { fired = append(fired, s) },
	}
	w.scan(context.Background())
	if got := repo.flipped[mid.String()+"/blue"]; !got {
		t.Fatalf("expected slot to be flipped to AI control")
	}
	if len(fired) != 1 || fired[0].Slot != "blue" {
		t.Fatalf("OnAITakeover hook missed: %+v", fired)
	}
}

func TestScanIsIdempotent(t *testing.T) {
	repo := newFakeRepo()
	mid := uuid.New()
	repo.pending = []pgrepo.InactiveSlot{{MatchID: mid, UserID: uuid.New(), Slot: "red"}}
	repo.flipped[mid.String()+"/red"] = true // pretend it's already flipped

	hits := 0
	w := &PresenceWorker{
		Logger: slog.Default(), Repo: repo,
		TakeoverAfter: 5 * time.Second,
		ScanEvery:     time.Hour,
		OnAITakeover:  func(_ context.Context, _ pgrepo.InactiveSlot) { hits++ },
	}
	w.scan(context.Background())
	if hits != 0 {
		t.Fatalf("expected hook to be skipped on no-op flip, got %d hits", hits)
	}
}

func TestScanLogsRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.listErr = errors.New("boom")
	w := &PresenceWorker{
		Logger:        slog.Default(),
		Repo:          repo,
		TakeoverAfter: time.Second,
		ScanEvery:     time.Hour,
	}
	w.scan(context.Background()) // must not panic
}
