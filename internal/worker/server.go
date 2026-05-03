package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/health"
	"github.com/joaquing/clone-supremacy/internal/platform/httpserver"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/service/notifysvc"
	"github.com/joaquing/clone-supremacy/internal/storage"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error {
	pool, err := storage.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := storage.Migrate(ctx, pool); err != nil {
		return err
	}

	nc, _, err := storage.NewNATS(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	repo := pgrepo.NewMatches(pool)
	users := pgrepo.NewUsers(pool)
	notifyRepo := pgrepo.NewNotifications(pool)
	notifyService := notifysvc.New(notifyRepo)

	snap := &SnapshotWorker{
		Logger:   logger,
		Metrics:  mreg,
		Repo:     repo,
		NC:       nc,
		Interval: cfg.SnapshotInterval,
	}
	go snap.Run(ctx) //nolint:gosec // ctx is the binary-mode shutdown context.

	presence := &PresenceWorker{
		Logger:        logger,
		NC:            nc,
		Repo:          repo,
		TakeoverAfter: cfg.AITakeoverAfter,
		ScanEvery:     cfg.AIPollInterval,
	}
	go presence.Run(ctx) //nolint:gosec // ditto.

	mailer := NewMailer(MailerConfig{
		Logger:   logger,
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		LookupTo: func(ctx context.Context, userID uuid.UUID) (string, error) {
			u, err := users.ByID(ctx, userID)
			if err != nil {
				return "", err
			}
			return u.Email, nil
		},
	})
	notify := &NotifyWorker{
		Logger:    logger,
		NC:        nc,
		Notify:    notifyService,
		Players:   matchPlayerLookup{Repo: repo},
		Mailer:    mailer,
		BatchSize: cfg.NotifyBatchSize,
		Poll:      cfg.NotifyPollInterval,
	}
	go notify.Run(ctx) //nolint:gosec // ditto.

	h := health.New(cfg.Mode, version)
	h.RegisterReady(func() error {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			return err
		}
		if !nc.IsConnected() {
			return errors.New("nats disconnected")
		}
		return nil
	})

	r := chi.NewRouter()
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)

	return httpserver.Run(ctx, httpserver.Config{
		Addr:    cfg.HTTPAddr,
		Handler: r,
		Logger:  logger,
		Name:    "worker",
	})
}
