package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TicketBroker mints short-lived single-use credentials to upgrade an HTTP
// session to a WebSocket. The web client gets a JWT for normal API calls and
// trades it for a ticket immediately before opening the WS, so JWTs do not
// have to be passed in query strings or cookies on the WebSocket handshake.
type TicketBroker struct {
	redis *redis.Client
	ttl   time.Duration
}

type Ticket struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	IssuedAt    time.Time `json:"issued_at"`
}

func NewTicketBroker(r *redis.Client, ttl time.Duration) *TicketBroker {
	return &TicketBroker{redis: r, ttl: ttl}
}

func (b *TicketBroker) Mint(ctx context.Context, t Ticket) (string, error) {
	id, err := randomID(24)
	if err != nil {
		return "", err
	}
	t.IssuedAt = time.Now()
	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	key := ticketKey(id)
	if err := b.redis.Set(ctx, key, payload, b.ttl).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}
	return id, nil
}

// Redeem consumes the ticket atomically; it can only be redeemed once.
func (b *TicketBroker) Redeem(ctx context.Context, id string) (Ticket, error) {
	key := ticketKey(id)
	raw, err := b.redis.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Ticket{}, ErrTicketInvalid
		}
		return Ticket{}, err
	}
	var t Ticket
	if err := json.Unmarshal(raw, &t); err != nil {
		return Ticket{}, ErrTicketInvalid
	}
	return t, nil
}

func ticketKey(id string) string { return "ws:ticket:" + id }

func randomID(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
