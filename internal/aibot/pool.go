package aibot

import (
	"sync"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/pkg/api"
)

// botPool round-robins through the seed pool of bot accounts so the
// same human player rarely gets the same bot identity twice in a row.
// Pre-seeded bot identities live in the `users` table (created by the
// `00003_phase5.sql` migration); the runner re-loads the pool on every
// scan so newly seeded bots get picked up without restarting.
type botPool struct {
	mu   sync.Mutex
	bots []api.BotUserView
	next int
}

func newBotPool(bots []api.BotUserView) *botPool {
	return &botPool{bots: append([]api.BotUserView{}, bots...)}
}

// take returns the next bot id, advancing the cursor. Returns
// ErrNoBotsAvailable when the pool is empty.
func (p *botPool) take() (uuid.UUID, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.bots) == 0 {
		return uuid.Nil, ErrNoBotsAvailable
	}
	b := p.bots[p.next%len(p.bots)]
	p.next++
	return b.ID, nil
}

// reload swaps the pool with a fresh list. Called by the runner on the
// scan cadence so newly seeded bots are picked up automatically.
func (p *botPool) reload(bots []api.BotUserView) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bots = append([]api.BotUserView{}, bots...)
	if len(p.bots) > 0 {
		p.next %= len(p.bots)
	} else {
		p.next = 0
	}
}

// size returns the number of bots in the pool. Useful for log output
// and the readiness probe.
func (p *botPool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.bots)
}
