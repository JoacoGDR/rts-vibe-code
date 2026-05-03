package chatsvc

import (
	"time"

	"github.com/google/uuid"
)

// tokenBucket is a tiny per-user rate limiter. We do not bother with
// Redis here — chat traffic is small and core-api is single-instance for
// MVP. When we shard core-api this becomes a Redis Lua script or moves
// to the gateway.
type tokenBucket struct {
	tokens    float64
	cap       float64
	refillPer float64 // tokens per second
	last      time.Time
}

func newBucket(rate float64) *tokenBucket {
	return &tokenBucket{tokens: rate, cap: rate, refillPer: rate, last: time.Now()}
}

func (b *tokenBucket) take() bool {
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.refillPer
	if b.tokens > b.cap {
		b.tokens = b.cap
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// allow records the message attempt and returns true when it is below
// the per-user rate.
func (s *Service) allow(userID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.throttle[userID]
	if !ok {
		b = newBucket(MessagesPerSecond)
		s.throttle[userID] = b
	}
	return b.take()
}
