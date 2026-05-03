package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/mw"
	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	v1 "github.com/joaquing/clone-supremacy/internal/adapter/httpapi/v1"
	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/pgrepo"
	"github.com/joaquing/clone-supremacy/internal/auth"
	"github.com/joaquing/clone-supremacy/internal/config"
	"github.com/joaquing/clone-supremacy/internal/platform/health"
	"github.com/joaquing/clone-supremacy/internal/platform/httpserver"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/internal/service/authsvc"
	"github.com/joaquing/clone-supremacy/internal/service/chatsvc"
	"github.com/joaquing/clone-supremacy/internal/service/lobbysvc"
	"github.com/joaquing/clone-supremacy/internal/service/notifysvc"
	"github.com/joaquing/clone-supremacy/internal/storage"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// coreAPIInfra bundles the long-lived clients core-api opens at startup.
// Consolidating them into a struct keeps RunCoreAPI's body short.
type coreAPIInfra struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	nc   *nats.Conn
	js   jetstream.JetStream
}

// RunCoreAPI is the composition root for the core-api binary mode. It
// wires Postgres / Redis / NATS adapters into the auth + lobby services,
// mounts the v1 controllers behind chi, and starts the HTTP server with
// the platform's graceful-shutdown helper.
func RunCoreAPI(ctx context.Context, cfg config.Config, logger *slog.Logger, mreg *metrics.Registry, version string) error {
	infra, err := connectCoreAPIInfra(ctx, cfg)
	if err != nil {
		return err
	}
	defer infra.pool.Close()
	defer func() { _ = infra.rdb.Close() }()
	defer infra.nc.Close()

	issuer := auth.NewIssuer(cfg.JWTSecret, cfg.JWTAccessTTL)
	tickets := auth.NewTicketBroker(infra.rdb, cfg.WSTicketTTL)
	publisher := &natsbridge.Publisher{JS: infra.js, NC: infra.nc}

	matchesRepo := pgrepo.NewMatches(infra.pool)
	authService := authsvc.New(pgrepo.NewUsers(infra.pool), issuer, tickets)
	lobbyService := lobbysvc.New(matchesRepo, publisher, cfg.GameTimeFactor)
	chatService := chatsvc.New(pgrepo.NewChat(infra.pool), matchesRepo, publisher)
	notifyService := notifysvc.New(pgrepo.NewNotifications(infra.pool))

	if err := startChatIngress(ctx, logger, infra.nc, chatService); err != nil {
		return err
	}

	router := newCoreAPIRouter(coreAPIDeps{
		Logger:    logger,
		Metrics:   mreg,
		Health:    newCoreAPIHealth(cfg, version, infra),
		Issuer:    issuer,
		Auth:      v1.NewAuthController(authService),
		Match:     v1.NewMatchController(lobbyService),
		Maps:      v1.NewMapController(),
		Chat:      v1.NewChatController(chatService),
		Notify:    v1.NewNotificationController(notifyService),
		BotAPIKey: cfg.BotAPIKey,
		Version:   version,
	})

	return httpserver.Run(ctx, httpserver.Config{
		Addr:    cfg.HTTPAddr,
		Handler: router,
		Logger:  logger,
		Name:    "core-api",
	})
}

// startChatIngress wires the NATS chat ingress subscription onto the
// chat service. The gateway publishes user-authored messages on
// chat.<matchID>.ingress; we validate, persist and re-publish to the
// scope's fanout subject from here.
func startChatIngress(ctx context.Context, logger *slog.Logger, nc *nats.Conn, svc *chatsvc.Service) error {
	return natsbridge.SubscribeChatIngress(ctx, logger, nc, func(reqCtx context.Context, p natsbridge.ChatIngressPayload) error {
		matchID, err := uuid.Parse(p.MatchID)
		if err != nil {
			return err
		}
		authorID, err := uuid.Parse(p.AuthorID)
		if err != nil {
			return err
		}
		_, err = svc.Send(reqCtx, chatsvc.SendInput{
			MatchID:    matchID,
			AuthorID:   authorID,
			AuthorSlot: p.AuthorSlot,
			Scope:      p.Scope,
			Body:       p.Body,
		})
		return err
	})
}

// connectCoreAPIInfra opens Postgres (and runs migrations), Redis and
// NATS in the order the rest of the service expects.
func connectCoreAPIInfra(ctx context.Context, cfg config.Config) (coreAPIInfra, error) {
	pool, err := storage.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return coreAPIInfra{}, err
	}
	if err := storage.Migrate(ctx, pool); err != nil {
		pool.Close()
		return coreAPIInfra{}, err
	}
	rdb, err := storage.NewRedis(cfg.RedisURL)
	if err != nil {
		pool.Close()
		return coreAPIInfra{}, err
	}
	nc, js, err := storage.NewNATS(cfg.NATSURL)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return coreAPIInfra{}, err
	}
	return coreAPIInfra{pool: pool, rdb: rdb, nc: nc, js: js}, nil
}

// newCoreAPIHealth builds the health probe with readiness checks that
// ping every dependency RunCoreAPI opened.
func newCoreAPIHealth(cfg config.Config, version string, infra coreAPIInfra) *health.Handler {
	h := health.New(cfg.Mode, version)
	h.RegisterReady(func() error {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := infra.pool.Ping(pingCtx); err != nil {
			return err
		}
		if err := infra.rdb.Ping(pingCtx).Err(); err != nil {
			return err
		}
		if !infra.nc.IsConnected() {
			return errors.New("nats disconnected")
		}
		return nil
	})
	return h
}

// coreAPIDeps bundles the things the chi router needs so we don't have
// to thread eight args into a constructor.
type coreAPIDeps struct {
	Logger    *slog.Logger
	Metrics   *metrics.Registry
	Health    *health.Handler
	Issuer    *auth.Issuer
	Auth      *v1.AuthController
	Match     *v1.MatchController
	Maps      *v1.MapController
	Chat      *v1.ChatController
	Notify    *v1.NotificationController
	BotAPIKey string
	Version   string
}

// newCoreAPIRouter constructs the chi mux with global middleware, health
// probes and the /api/v1 routes.
func newCoreAPIRouter(d coreAPIDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(mw.Logging(d.Logger))
	r.Use(mw.Metrics(d.Metrics))
	r.Use(mw.CORS)

	r.Get("/healthz", d.Health.Live)
	r.Get("/readyz", d.Health.Ready)
	r.Route("/api/v1", func(r chi.Router) { mountV1(r, d) })

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		render.Err(w, errs.New(errs.NotFound, "route not found"))
	})
	return r
}

// mountV1 registers the public REST surface. Public endpoints first;
// authenticated endpoints inside the chi.Group.
func mountV1(r chi.Router, d coreAPIDeps) {
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(d.Version))
	})
	r.Get("/maps", d.Maps.List)
	r.Post("/auth/register", d.Auth.Register)
	r.Post("/auth/login", d.Auth.Login)

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireBotKey(d.BotAPIKey))
		r.Post("/auth/login-as-bot", d.Auth.LoginAsBot)
		r.Get("/auth/bots", d.Auth.ListBots)
	})

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth(d.Issuer))
		r.Post("/auth/ws-ticket", d.Auth.WSTicket)
		r.Get("/auth/me", d.Auth.Me)
		r.Post("/matches", d.Match.Create)
		r.Get("/matches", d.Match.List)
		r.Get("/matches/{matchID}", d.Match.Get)
		r.Post("/matches/{matchID}/join", d.Match.Join)
		r.Post("/matches/{matchID}/start", d.Match.Start)
		r.Post("/matches/{matchID}/chat", d.Chat.Send)
		r.Get("/matches/{matchID}/chat", d.Chat.History)
		r.Get("/notifications", d.Notify.List)
		r.Get("/notifications/unread-count", d.Notify.UnreadCount)
		r.Post("/notifications/mark-read", d.Notify.MarkRead)
	})
}
