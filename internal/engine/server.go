package engine

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/redisrepo"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/domain/cmddom"
	"github.com/joaquing/clone-supremacy/internal/domain/ids"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/platform/health"
	"github.com/joaquing/clone-supremacy/internal/platform/httpserver"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/service/enginesvc"
	"github.com/joaquing/clone-supremacy/internal/storage"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error {
	nc, js, err := storage.NewNATS(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	rdb, err := storage.NewRedis(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	if err := natsbridge.EnsureStreams(ctx, js); err != nil {
		return err
	}

	publisher := &natsbridge.Publisher{JS: js, NC: nc}
	slots := &redisrepo.SlotIndex{Client: rdb, TTL: 24 * time.Hour}
	runner := enginesvc.NewRunner(logger, mreg, publisher, slots)

	if err := subscribeEngine(ctx, logger, nc, js, runner); err != nil {
		return err
	}

	r := chi.NewRouter()
	h := health.New(cfg.Mode, version)
	h.RegisterReady(func() error {
		if !nc.IsConnected() {
			return errors.New("nats not connected")
		}
		return nil
	})
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)

	return httpserver.Run(ctx, httpserver.Config{
		Addr:    cfg.HTTPAddr,
		Handler: r,
		Logger:  logger,
		Name:    "engine",
	})
}

// subscribeEngine wires the start/command/resync handlers onto NATS and
// hands incoming payloads to the engine runner. Pulled out of Run so the
// composition root stays under the funlen budget.
func subscribeEngine(ctx context.Context, logger *slog.Logger, nc *nats.Conn, js jetstream.JetStream, runner *enginesvc.Runner) error {
	startHandler := makeStartHandler(ctx, logger, runner)
	commandHandler := makeCommandHandler(runner)
	resyncHandler := func(p natsbridge.ResyncPayload) { runner.Rebroadcast(ctx, p.MatchID) }

	if err := natsbridge.SubscribeStarts(ctx, logger, js, startHandler); err != nil {
		return err
	}
	if err := natsbridge.SubscribeCommands(ctx, logger, js, commandHandler); err != nil {
		return err
	}
	if err := natsbridge.SubscribeResyncs(ctx, logger, nc, resyncHandler); err != nil {
		return err
	}
	endHandler := func(p natsbridge.EndMatchPayload) error {
		runner.ForceEnd(ctx, p.MatchID)
		return nil
	}
	return natsbridge.SubscribeEndMatches(ctx, logger, nc, endHandler)
}

func makeStartHandler(ctx context.Context, logger *slog.Logger, runner *enginesvc.Runner) func(natsbridge.StartPayload) error {
	return func(p natsbridge.StartPayload) error {
		match, err := matchdom.New(p.MatchID, p.MapID, p.SlotAssignments, p.Speed, p.StartedAt)
		if err != nil {
			logger.Error("could not build match", "err", err, "match", p.MatchID)
			return err
		}
		for slot, userID := range p.SlotAssignments {
			if err := runner.SlotsIndex().SetUserSlot(p.MatchID, userID, slot); err != nil {
				logger.Warn("slot index write failed", "err", err, "user", userID)
			}
		}
		runner.EnsureMatch(ctx, match)
		runner.BroadcastInitial(ctx, match)
		return nil
	}
}

func makeCommandHandler(runner *enginesvc.Runner) func(natsbridge.CommandPayload) error {
	return func(p natsbridge.CommandPayload) error {
		cmd := cmddom.Command{
			MatchID:        ids.MatchID(p.MatchID),
			UserID:         ids.UserID(p.UserID),
			IssuerSlot:     ids.SlotID(p.Slot),
			IdempotencyKey: p.IdempotencyKey,
			Kind:           p.Kind,
			UnitID:         ids.UnitID(p.UnitID),
			From:           ids.ProvinceID(p.From),
			To:             ids.ProvinceID(p.To),
			IssuedAt:       p.IssuedAt,
			Args:           p.Args,
		}
		if err := runner.Submit(cmd); err != nil {
			if errors.Is(err, enginesvc.ErrNotHosted) {
				return natsbridge.ErrNotHosted
			}
			return err
		}
		return nil
	}
}
