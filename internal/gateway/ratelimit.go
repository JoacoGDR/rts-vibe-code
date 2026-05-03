package gateway

import (
	"sync"
	"time"
)

// tokenBucket is a tiny per-connection rate limiter. Refills `capacity`
// tokens every `refill` interval.
type tokenBucket struct {
	mu       sync.Mutex
	capacity int
	tokens   int
	last     time.Time
	refill   time.Duration
}

func newTokenBucket(capacity int, refill time.Duration) *tokenBucket {
	return &tokenBucket{capacity: capacity, tokens: capacity, last: time.Now(), refill: refill}
}

func (b *tokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if now.Sub(b.last) >= b.refill {
		b.tokens = b.capacity
		b.last = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}
