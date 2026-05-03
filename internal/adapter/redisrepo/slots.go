package redisrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SlotIndex stores the user->slot assignment in Redis for the gateway
// to look up at WebSocket connection time. Keys are namespaced by match.
type SlotIndex struct {
	Client *redis.Client
	TTL    time.Duration
}

// SetUserSlot stores a single user->slot entry. Empty userID/slot are
// silently skipped so callers can pass partial lobbies without surprises.
func (r *SlotIndex) SetUserSlot(matchID, userID, slot string) error {
	if userID == "" || slot == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ttl := r.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return r.Client.Set(ctx, slotKey(matchID, userID), slot, ttl).Err()
}

// LookupUserSlot resolves a user's slot in a match. Returns "" if missing.
func LookupUserSlot(ctx context.Context, c *redis.Client, matchID, userID string) (string, error) {
	v, err := c.Get(ctx, slotKey(matchID, userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func slotKey(matchID, userID string) string {
	return fmt.Sprintf("match:%s:user:%s:slot", matchID, userID)
}
