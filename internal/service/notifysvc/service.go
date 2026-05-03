package notifysvc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// Repository is the persistence dependency. The Postgres implementation
// lives in [pgrepo.Notifications].
type Repository interface {
	Insert(ctx context.Context, n notifydom.Notification) (notifydom.Notification, error)
	LastForKind(ctx context.Context, userID, matchID uuid.UUID, kind notifydom.Kind) (notifydom.Notification, error)
	List(ctx context.Context, q notifydom.ListQuery) ([]notifydom.Notification, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error)
	PendingDelivery(ctx context.Context, limit int) ([]notifydom.Notification, error)
	MarkSent(ctx context.Context, id uuid.UUID, at time.Time) error
}

// Service is the use-case orchestrator. Construct one per process and
// share — there is no per-request mutable state.
type Service struct {
	repo Repository
}

func New(repo Repository) *Service { return &Service{repo: repo} }

// Insert persists a notification, honouring the per-kind debounce
// window. Returns (zero, nil) when a duplicate would have landed inside
// the debounce window — the caller can use the zero ID to detect a
// silent skip.
type Insert struct {
	UserID  uuid.UUID
	MatchID uuid.UUID
	Kind    notifydom.Kind
	Payload map[string]any
}

// Insert is the worker-side entry point. Every consumer should go
// through this so the debounce policy stays in one place.
func (s *Service) Insert(ctx context.Context, in Insert) (notifydom.Notification, error) {
	if in.UserID == uuid.Nil || in.MatchID == uuid.Nil || in.Kind == "" {
		return notifydom.Notification{}, errs.New(errs.BadRequest, "missing notification fields")
	}
	window := in.Kind.DebounceWindow()
	if window > 0 {
		last, err := s.repo.LastForKind(ctx, in.UserID, in.MatchID, in.Kind)
		switch {
		case err == nil && time.Since(last.CreatedAt) < window:
			return notifydom.Notification{}, nil
		case err != nil && !errors.Is(err, notifydom.ErrNotFound):
			return notifydom.Notification{}, errs.Wrap(err, errs.Internal, "checking debounce")
		}
	}
	n, err := s.repo.Insert(ctx, notifydom.Notification{
		UserID: in.UserID, MatchID: in.MatchID, Kind: in.Kind, Payload: in.Payload,
	})
	if err != nil {
		return notifydom.Notification{}, errs.Wrap(err, errs.Internal, "inserting notification")
	}
	return n, nil
}

// List returns the user's most recent notifications.
func (s *Service) List(ctx context.Context, q notifydom.ListQuery) ([]notifydom.Notification, error) {
	out, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, errs.Wrap(err, errs.Internal, "listing notifications")
	}
	return out, nil
}

// UnreadCount is the lightweight bell-badge query.
func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return 0, errs.Wrap(err, errs.Internal, "counting notifications")
	}
	return n, nil
}

// MarkAllRead is invoked by the bell when the user opens it.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := s.repo.MarkAllRead(ctx, userID, time.Now())
	if err != nil {
		return 0, errs.Wrap(err, errs.Internal, "marking notifications read")
	}
	return n, nil
}

// PendingDelivery is the SMTP worker's "pull next batch" call.
func (s *Service) PendingDelivery(ctx context.Context, limit int) ([]notifydom.Notification, error) {
	out, err := s.repo.PendingDelivery(ctx, limit)
	if err != nil {
		return nil, errs.Wrap(err, errs.Internal, "loading pending notifications")
	}
	return out, nil
}

// MarkSent stamps a delivery success. Failures bubble up so the worker
// can decide to retry or surface the error.
func (s *Service) MarkSent(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.MarkSent(ctx, id, time.Now()); err != nil {
		return errs.Wrap(err, errs.Internal, "marking notification sent")
	}
	return nil
}
