package enginesvc

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

// TestRunnerRebroadcastNoRace hammers resync while the match loop ticks and
// reads m.Units for visibility. Without routing Rebroadcast through the match
// loop this fails go test -race with concurrent map iteration/write on m.Units.
func TestRunnerRebroadcastNoRace(t *testing.T) {
	now := time.Now()
	m, err := matchdom.New("race-resync", "tiny-2p",
		map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatal(err)
	}

	r := NewRunner(slog.Default(), nil, noopBroadcaster{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.EnsureMatch(ctx, m)

	const workers = 8
	const iters = 200
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iters {
				r.Rebroadcast(ctx, m.ID)
			}
		}()
	}
	wg.Wait()
}
