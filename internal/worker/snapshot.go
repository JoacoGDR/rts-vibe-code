package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// SnapshotWorker subscribes to every match's state subject and writes the
// latest seen state to Postgres at a fixed cadence. It is deliberately lossy:
// the JetStream command log is the WAL, this is a recovery point.
type SnapshotWorker struct {
	Logger   *slog.Logger
	Metrics  *metrics.Registry
	Repo     *pgrepo.Matches
	NC       *nats.Conn
	Interval time.Duration

	mu     sync.Mutex
	latest map[uuid.UUID]wire.ServerEnvelope
}

func (s *SnapshotWorker) Run(ctx context.Context) {
	if s.Interval == 0 {
		s.Interval = 5 * time.Minute
	}
	s.latest = map[uuid.UUID]wire.ServerEnvelope{}

	sub, err := s.NC.Subscribe("match.*.state", func(m *nats.Msg) {
		var env wire.ServerEnvelope
		if err := json.Unmarshal(m.Data, &env); err != nil {
			return
		}
		mid, err := uuid.Parse(env.MatchID)
		if err != nil {
			return
		}
		s.mu.Lock()
		s.latest[mid] = env
		s.mu.Unlock()
	})
	if err != nil {
		s.Logger.Error("subscribe state failed", "err", err)
		return
	}
	defer func() { _ = sub.Unsubscribe() }()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flush(context.Background())
			return
		case <-ticker.C:
			s.flush(ctx)
		}
	}
}

func (s *SnapshotWorker) flush(ctx context.Context) {
	s.mu.Lock()
	pending := s.latest
	s.latest = map[uuid.UUID]wire.ServerEnvelope{}
	s.mu.Unlock()
	if len(pending) == 0 {
		return
	}
	for matchID, env := range pending {
		if env.State == nil {
			continue
		}
		if err := s.Repo.SaveSnapshot(ctx, matchID, env.Seq, env); err != nil {
			s.Logger.Warn("snapshot save failed", "match", matchID, "err", err)
			continue
		}
		s.Logger.Info("snapshot saved", "match", matchID, "seq", env.Seq)
	}
}
