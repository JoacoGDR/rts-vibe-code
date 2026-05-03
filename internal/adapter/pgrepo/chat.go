package pgrepo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
)

// Chat is the Postgres-backed implementation of [chatsvc.MessagesRepository].
type Chat struct {
	pool *pgxpool.Pool
}

func NewChat(pool *pgxpool.Pool) *Chat { return &Chat{pool: pool} }

// Insert persists a single message. The id, sent_at and other fields are
// taken from the supplied [chatdom.Message] so the service can stamp them
// consistently with the NATS publish payload.
func (r *Chat) Insert(ctx context.Context, m chatdom.Message) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO chat_messages (id, match_id, scope, author_user_id, author_slot, body, sent_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.MatchID, m.Scope, m.AuthorID, m.AuthorSlot, m.Body, m.SentAt)
	return err
}

// History returns messages in the given scope ordered newest-first. The
// caller is responsible for reversing them when rendering chronologically.
func (r *Chat) History(ctx context.Context, q chatdom.HistoryQuery) ([]chatdom.Message, error) {
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	beforeArg := q.Before
	if beforeArg.IsZero() {
		beforeArg = time.Now().Add(time.Hour) // sentinel: "any time before the future"
	}
	rows, err := r.pool.Query(ctx, `
        SELECT id, match_id, scope, author_user_id, author_slot, body, sent_at
        FROM chat_messages
        WHERE match_id = $1 AND scope = $2 AND sent_at < $3
        ORDER BY sent_at DESC
        LIMIT $4`,
		q.MatchID, q.Scope, beforeArg, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []chatdom.Message{}
	for rows.Next() {
		var m chatdom.Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.Scope, &m.AuthorID, &m.AuthorSlot, &m.Body, &m.SentAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
