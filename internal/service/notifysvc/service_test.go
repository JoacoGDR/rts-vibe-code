package notifysvc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

type stubRepo struct {
	mu       sync.Mutex
	rows     []notifydom.Notification
	pending  []notifydom.Notification
	insertCt int
}

func (r *stubRepo) Insert(_ context.Context, n notifydom.Notification) (notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.insertCt++
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	r.rows = append(r.rows, n)
	r.pending = append(r.pending, n)
	return n, nil
}

func (r *stubRepo) LastForKind(_ context.Context, userID, matchID uuid.UUID, kind notifydom.Kind) (notifydom.Notification, error) {
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

func (r *stubRepo) List(_ context.Context, q notifydom.ListQuery) ([]notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []notifydom.Notification{}
	for _, n := range r.rows {
		if n.UserID == q.UserID {
			out = append(out, n)
		}
	}
	return out, nil
}

func (r *stubRepo) UnreadCount(_ context.Context, userID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, n := range r.rows {
		if n.UserID == userID && n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *stubRepo) MarkAllRead(_ context.Context, userID uuid.UUID, at time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for i := range r.rows {
		if r.rows[i].UserID == userID && r.rows[i].ReadAt == nil {
			t := at
			r.rows[i].ReadAt = &t
			n++
		}
	}
	return n, nil
}

func (r *stubRepo) PendingDelivery(_ context.Context, _ int) ([]notifydom.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]notifydom.Notification{}, r.pending...), nil
}

func (r *stubRepo) MarkSent(_ context.Context, id uuid.UUID, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.pending[:0]
	for _, n := range r.pending {
		if n.ID != id {
			out = append(out, n)
		}
	}
	r.pending = out
	return nil
}

func TestInsertRequiresFields(t *testing.T) {
	svc := New(&stubRepo{})
	_, err := svc.Insert(context.Background(), Insert{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if e := errs.As(err); e == nil || e.Code != errs.BadRequest {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestInsertDebouncesUnderAttack(t *testing.T) {
	repo := &stubRepo{}
	svc := New(repo)
	user, match := uuid.New(), uuid.New()
	in := Insert{UserID: user, MatchID: match, Kind: notifydom.KindUnderAttack}
	for i := 0; i < 4; i++ {
		if _, err := svc.Insert(context.Background(), in); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if repo.insertCt != 1 {
		t.Fatalf("debounce should have collapsed to 1 insert, got %d", repo.insertCt)
	}
}

func TestInsertDoesNotDebounceMatchEnded(t *testing.T) {
	repo := &stubRepo{}
	svc := New(repo)
	user, match := uuid.New(), uuid.New()
	for i := 0; i < 3; i++ {
		if _, err := svc.Insert(context.Background(), Insert{
			UserID: user, MatchID: match, Kind: notifydom.KindMatchEnded,
		}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if repo.insertCt != 3 {
		t.Fatalf("match_ended should never debounce, got %d", repo.insertCt)
	}
}

func TestMarkAllReadClearsBadge(t *testing.T) {
	repo := &stubRepo{}
	svc := New(repo)
	user, match := uuid.New(), uuid.New()
	if _, err := svc.Insert(context.Background(), Insert{
		UserID: user, MatchID: match, Kind: notifydom.KindMatchEnded,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	count, err := svc.UnreadCount(context.Background(), user)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 unread, got %d (err=%v)", count, err)
	}
	if _, err := svc.MarkAllRead(context.Background(), user); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, _ = svc.UnreadCount(context.Background(), user)
	if count != 0 {
		t.Fatalf("expected zero unread after mark-all-read, got %d", count)
	}
}
