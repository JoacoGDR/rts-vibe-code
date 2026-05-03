package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/internal/service/notifysvc"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// fakeNotifyRepo is a tiny in-memory notifysvc.Repository. It tracks
// inserts (with debounce) so tests can assert what landed.
type fakeNotifyRepo struct {
	mu       sync.Mutex
	rows     []notifydom.Notification
	pendings []notifydom.Notification
}

func newFakeNotifyRepo() *fakeNotifyRepo { return &fakeNotifyRepo{} }

func (r *fakeNotifyRepo) Insert(_ context.Context, n notifydom.Notification) (notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	r.rows = append(r.rows, n)
	r.pendings = append(r.pendings, n)
	return n, nil
}

func (r *fakeNotifyRepo) LastForKind(_ context.Context, userID, matchID uuid.UUID, kind notifydom.Kind) (notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if row.UserID == userID && row.MatchID == matchID && row.Kind == kind {
			return row, nil
		}
	}
	return notifydom.Notification{}, notifydom.ErrNotFound
}

func (r *fakeNotifyRepo) List(_ context.Context, _ notifydom.ListQuery) ([]notifydom.Notification, error) {
	return nil, nil
}

func (r *fakeNotifyRepo) UnreadCount(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }

func (r *fakeNotifyRepo) MarkAllRead(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *fakeNotifyRepo) PendingDelivery(_ context.Context, _ int) ([]notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]notifydom.Notification{}, r.pendings...)
	return out, nil
}

func (r *fakeNotifyRepo) MarkSent(_ context.Context, id uuid.UUID, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.pendings[:0]
	for _, n := range r.pendings {
		if n.ID == id {
			continue
		}
		out = append(out, n)
	}
	r.pendings = out
	return nil
}

// fakePlayers returns a static roster so tests can assert which slots
// got notified.
type fakePlayers struct {
	players []playerLookupResult
	err     error
}

func (f fakePlayers) ListPlayers(_ context.Context, _ uuid.UUID) ([]playerLookupResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.players, nil
}

// fakeMailer captures every Send so tests can assert delivery.
type fakeMailer struct {
	mu   sync.Mutex
	sent []notifydom.Notification
	err  error
}

func (m *fakeMailer) Send(_ context.Context, n notifydom.Notification) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, n)
	return nil
}

func TestNotifyWorkerInsertsForCaptureLoss(t *testing.T) {
	repo := newFakeNotifyRepo()
	svc := notifysvc.New(repo)
	red, blue := uuid.New(), uuid.New()
	w := &NotifyWorker{
		Logger: slog.Default(),
		Notify: svc,
		Players: fakePlayers{players: []playerLookupResult{
			{UserID: red, Slot: "red", Alive: true},
			{UserID: blue, Slot: "blue", Alive: true},
		}},
	}
	matchID := uuid.New()
	env := wire.ServerEnvelope{
		Type: wire.ServerEvent, MatchID: matchID.String(),
		SentAt: time.Now(),
		Event: &wire.Event{Kind: "province_captured", Province: "p1",
			Extra: map[string]any{"prev_owner": "red", "new_owner": "blue"}},
	}
	raw, _ := json.Marshal(env)
	w.handleEvent(context.Background(), raw)
	if len(repo.rows) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != red {
		t.Fatalf("expected to notify the previous owner (red), got %v", repo.rows[0].UserID)
	}
	if repo.rows[0].Kind != notifydom.KindProvinceCaptured {
		t.Fatalf("unexpected kind %s", repo.rows[0].Kind)
	}
}

func TestNotifyWorkerDebouncesUnderAttack(t *testing.T) {
	repo := newFakeNotifyRepo()
	svc := notifysvc.New(repo)
	red := uuid.New()
	w := &NotifyWorker{
		Logger: slog.Default(),
		Notify: svc,
		Players: fakePlayers{players: []playerLookupResult{
			{UserID: red, Slot: "red", Alive: true},
		}},
	}
	matchID := uuid.New()
	for i := 0; i < 5; i++ {
		env := wire.ServerEnvelope{
			Type: wire.ServerEvent, MatchID: matchID.String(),
			SentAt: time.Now(),
			Event: &wire.Event{Kind: "combat_damage", Province: "p1",
				Extra: map[string]any{"defender": "red", "attacker": "blue"}},
		}
		raw, _ := json.Marshal(env)
		w.handleEvent(context.Background(), raw)
	}
	if len(repo.rows) != 1 {
		t.Fatalf("debounce broken: expected 1 notification, got %d", len(repo.rows))
	}
}

func TestDispatchSendsAndMarksSent(t *testing.T) {
	repo := newFakeNotifyRepo()
	svc := notifysvc.New(repo)
	mailer := &fakeMailer{}
	w := &NotifyWorker{Logger: slog.Default(), Notify: svc, Mailer: mailer}

	if _, err := svc.Insert(context.Background(), notifysvc.Insert{
		UserID: uuid.New(), MatchID: uuid.New(),
		Kind: notifydom.KindMatchEnded, Payload: map[string]any{"winners": []string{"red"}},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	w.dispatch(context.Background())
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one mail sent, got %d", len(mailer.sent))
	}
	if len(repo.pendings) != 0 {
		t.Fatalf("pending queue should be empty after dispatch, got %d", len(repo.pendings))
	}
}

func TestDispatchSurvivesMailerError(t *testing.T) {
	repo := newFakeNotifyRepo()
	svc := notifysvc.New(repo)
	w := &NotifyWorker{
		Logger: slog.Default(), Notify: svc,
		Mailer: &fakeMailer{err: errors.New("smtp down")},
	}
	if _, err := svc.Insert(context.Background(), notifysvc.Insert{
		UserID: uuid.New(), MatchID: uuid.New(),
		Kind: notifydom.KindMatchEnded,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	w.dispatch(context.Background())
	if len(repo.pendings) != 1 {
		t.Fatalf("pending must remain when SMTP fails, got %d", len(repo.pendings))
	}
}

func TestNotifyWorkerSkipsAIControlled(t *testing.T) {
	repo := newFakeNotifyRepo()
	svc := notifysvc.New(repo)
	bot := uuid.New()
	w := &NotifyWorker{
		Logger: slog.Default(),
		Notify: svc,
		Players: fakePlayers{players: []playerLookupResult{
			{UserID: bot, Slot: "red", Alive: true, ControlledByAI: true},
		}},
	}
	matchID := uuid.New()
	env := wire.ServerEnvelope{
		Type: wire.ServerEvent, MatchID: matchID.String(),
		SentAt: time.Now(),
		Event: &wire.Event{Kind: "combat_damage", Extra: map[string]any{"defender": "red"}},
	}
	raw, _ := json.Marshal(env)
	w.handleEvent(context.Background(), raw)
	if len(repo.rows) != 0 {
		t.Fatalf("AI-controlled slots should not get notifications, got %d", len(repo.rows))
	}
}
