package pgrepo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
)

// Notifications is the Postgres-backed implementation of the
// notification persistence port. The worker uses it to enqueue and
// mark-as-sent; core-api uses List/MarkRead to back the bell endpoint.
type Notifications struct {
	pool *pgxpool.Pool
}

func NewNotifications(pool *pgxpool.Pool) *Notifications {
	return &Notifications{pool: pool}
}

// Insert persists a fresh notification. ID and CreatedAt are populated
// by the database when zero — keeps callers from having to stamp them.
func (r *Notifications) Insert(ctx context.Context, n notifydom.Notification) (notifydom.Notification, error) {
	payload, err := json.Marshal(n.Payload)
	if err != nil {
		return notifydom.Notification{}, err
	}
	row := r.pool.QueryRow(ctx, `
        INSERT INTO notifications (user_id, match_id, kind, payload)
        VALUES ($1, $2, $3, $4::jsonb)
        RETURNING id, user_id, match_id, kind, payload, created_at, sent_at, read_at`,
		n.UserID, n.MatchID, string(n.Kind), string(payload))
	return scanNotification(row)
}

// LastForKind returns the most recent notification of the given kind
// for the (user, match) pair. Used by the worker's debounce check; a
// missing row returns ErrNotFound.
func (r *Notifications) LastForKind(ctx context.Context, userID, matchID uuid.UUID, kind notifydom.Kind) (notifydom.Notification, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, user_id, match_id, kind, payload, created_at, sent_at, read_at
        FROM notifications
        WHERE user_id = $1 AND match_id = $2 AND kind = $3
        ORDER BY created_at DESC LIMIT 1`,
		userID, matchID, string(kind))
	n, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return notifydom.Notification{}, notifydom.ErrNotFound
	}
	return n, err
}

// List returns notifications for a user in newest-first order, optionally
// filtered to "after" (exclusive). Limit defaults to 50, capped at 200.
func (r *Notifications) List(ctx context.Context, q notifydom.ListQuery) ([]notifydom.Notification, error) {
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	after := q.After
	if after.IsZero() {
		after = time.Unix(0, 0)
	}
	rows, err := r.pool.Query(ctx, `
        SELECT id, user_id, match_id, kind, payload, created_at, sent_at, read_at
        FROM notifications
        WHERE user_id = $1 AND created_at > $2
        ORDER BY created_at DESC
        LIMIT $3`,
		q.UserID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notifydom.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// UnreadCount is a fast count query for the bell badge.
func (r *Notifications) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM notifications
        WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkAllRead stamps every unread notification for the user as read at
// the given instant. Returns the number of rows touched.
func (r *Notifications) MarkAllRead(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
        UPDATE notifications SET read_at = $2
        WHERE user_id = $1 AND read_at IS NULL`, userID, at)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// MarkSent stamps the given notification as delivered to the email
// transport (or fake-delivered when SMTP is unconfigured).
func (r *Notifications) MarkSent(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET sent_at = $2 WHERE id = $1`, id, at)
	return err
}

// PendingDelivery returns notifications waiting to be sent over SMTP.
// Limit caps how many we pick per worker tick.
func (r *Notifications) PendingDelivery(ctx context.Context, limit int) ([]notifydom.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
        SELECT id, user_id, match_id, kind, payload, created_at, sent_at, read_at
        FROM notifications
        WHERE sent_at IS NULL
        ORDER BY created_at
        LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notifydom.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNotification(row rowScanner) (notifydom.Notification, error) {
	var (
		n       notifydom.Notification
		payload []byte
		kind    string
	)
	if err := row.Scan(
		&n.ID, &n.UserID, &n.MatchID, &kind, &payload,
		&n.CreatedAt, &n.SentAt, &n.ReadAt,
	); err != nil {
		return notifydom.Notification{}, err
	}
	n.Kind = notifydom.Kind(kind)
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &n.Payload)
	}
	return n, nil
}
