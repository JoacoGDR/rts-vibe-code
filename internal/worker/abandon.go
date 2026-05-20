package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

// AbandonRepo is the DB surface for abandonment scans.
type AbandonRepo interface {
	ActiveMatches(ctx context.Context) ([]lobbydom.Match, error)
	MatchAbandonedCandidate(ctx context.Context, matchID uuid.UUID, thresholdSeconds int64) (bool, error)
	MarkAbandoned(ctx context.Context, id uuid.UUID) error
}

// AbandonWorker periodically marks silent active matches abandoned and
// asks the engine to force-end them.
type AbandonWorker struct {
	Logger    *slog.Logger
	NC        *nats.Conn
	Repo      AbandonRepo
	Threshold time.Duration
	ScanEvery time.Duration
}

func (w *AbandonWorker) Run(ctx context.Context) {
	if w.Threshold <= 0 {
		w.Threshold = 30 * time.Minute
	}
	if w.ScanEvery <= 0 {
		w.ScanEvery = time.Minute
	}
	t := time.NewTicker(w.ScanEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.scan(ctx)
		}
	}
}

func (w *AbandonWorker) scan(ctx context.Context) {
	ms, err := w.Repo.ActiveMatches(ctx)
	if err != nil {
		if w.Logger != nil {
			w.Logger.Warn("abandon list failed", "err", err)
		}
		return
	}
	sec := int64(w.Threshold.Seconds())
	pub := natsbridge.Publisher{NC: w.NC}
	for _, m := range ms {
		ok, err := w.Repo.MatchAbandonedCandidate(ctx, m.ID, sec)
		if err != nil || !ok {
			continue
		}
		if err := w.Repo.MarkAbandoned(ctx, m.ID); err != nil {
			if w.Logger != nil {
				w.Logger.Warn("mark abandoned failed", "match", m.ID, "err", err)
			}
			continue
		}
		if w.NC != nil {
			if err := pub.PublishEndMatch(ctx, natsbridge.EndMatchPayload{
				MatchID: m.ID.String(), Reason: "abandoned",
			}); err != nil && w.Logger != nil {
				w.Logger.Warn("end-match publish failed", "match", m.ID, "err", err)
			}
		}
		if w.Logger != nil {
			w.Logger.Info("match abandoned", "match", m.ID)
		}
	}
}

// Ensure pgrepo.Matches satisfies AbandonRepo.
var _ AbandonRepo = (*pgrepo.Matches)(nil)
