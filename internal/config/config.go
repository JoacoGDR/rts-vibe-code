// Package config centralises environment-driven configuration for every binary mode.
// All knobs are read once at startup so individual subsystems do not reach into
// os.Getenv directly.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Mode        string
	Env         string
	LogLevel    string
	LogFormat   string
	HTTPAddr    string
	MetricsAddr string

	DatabaseURL string
	RedisURL    string
	NATSURL     string

	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration
	WSTicketTTL    time.Duration
	CommandRateMax int

	SnapshotInterval time.Duration
	MacroPulseEvery  time.Duration

	GameTimeFactor float64

	// Phase 5: AI takeover knobs. Presence is stamped only when a user
	// opens a match WebSocket — there is no periodic heartbeat — so the
	// threshold is intentionally measured in days.
	BotAPIKey       string        // shared secret protecting POST /auth/login-as-bot
	AITakeoverAfter time.Duration // how long a slot must be silent before flipping to AI (default 72h)
	AIPollInterval  time.Duration // worker: takeover scan cadence (default 1h); ai-bot: bot pool/scan cadence (override to 30s in compose)
	AIBotCoreAPIURL string        // ai-bot mode: base URL of core-api
	AIBotGatewayURL string        // ai-bot mode: base URL of gateway (for ws://...)

	// Phase 6: abandonment when every alive human slot is silent.
	AbandonAfter        time.Duration // wall-clock silence before abandoning (default 30m)
	AbandonScanInterval time.Duration // worker scan cadence (default 1m)

	// Phase 5: notification + SMTP knobs. Empty SMTPHost means dev mode
	// (notifications are still persisted, but the worker logs and stamps
	// sent_at without dialling out).
	NotifyPollInterval time.Duration
	NotifyBatchSize    int
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	SMTPFrom           string

	// MapAssetsBaseURL is the CDN/S3 origin for province SVGs (e.g.
	// https://assets.example.com). When empty, core-api returns relative
	// paths (/maps/{id}.svg) for the SPA to serve from its static host.
	MapAssetsBaseURL string
}

const (
	ModeCoreAPI = "core-api"
	ModeGateway = "gateway"
	ModeEngine  = "engine"
	ModeWorker  = "worker"
	ModeAIBot   = "ai-bot"
)

func Load(mode string) (Config, error) {
	cfg := Config{
		Mode:             mode,
		Env:              getenv("ENV", "dev"),
		LogLevel:         getenv("LOG_LEVEL", "info"),
		LogFormat:        getenv("LOG_FORMAT", "json"),
		HTTPAddr:         getenv("HTTP_ADDR", defaultHTTPAddr(mode)),
		MetricsAddr:      getenv("METRICS_ADDR", ":9100"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://supremacy:supremacy@localhost:5432/supremacy?sslmode=disable"),
		RedisURL:         getenv("REDIS_URL", "redis://localhost:6379/0"),
		NATSURL:          getenv("NATS_URL", "nats://localhost:4222"),
		JWTSecret:        getenv("JWT_SECRET", "dev-only-secret-change-me"),
		JWTAccessTTL:     getDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:    getDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		WSTicketTTL:      getDuration("WS_TICKET_TTL", 30*time.Second),
		CommandRateMax:   getInt("COMMAND_RATE_MAX", 30),
		SnapshotInterval: getDuration("SNAPSHOT_INTERVAL", 5*time.Minute),
		MacroPulseEvery:  getDuration("MACRO_PULSE_EVERY", 15*time.Minute),
		GameTimeFactor:   getFloat("GAME_TIME_FACTOR", 60),

		BotAPIKey:       getenv("BOT_API_KEY", "dev-only-bot-key-change-me"),
		AITakeoverAfter: getDuration("AI_TAKEOVER_AFTER", 72*time.Hour),
		AIPollInterval:  getDuration("AI_POLL_INTERVAL", time.Hour),
		AIBotCoreAPIURL: getenv("AI_BOT_CORE_API_URL", "http://localhost:8080"),
		AIBotGatewayURL: getenv("AI_BOT_GATEWAY_URL", "ws://localhost:8081"),

		AbandonAfter:        getDuration("ABANDON_AFTER", 30*time.Minute),
		AbandonScanInterval: getDuration("ABANDON_SCAN_INTERVAL", time.Minute),

		NotifyPollInterval: getDuration("NOTIFY_POLL_INTERVAL", 15*time.Second),
		NotifyBatchSize:    getInt("NOTIFY_BATCH_SIZE", 50),
		SMTPHost:           getenv("SMTP_HOST", ""),
		SMTPPort:           getInt("SMTP_PORT", 1025),
		SMTPUsername:       getenv("SMTP_USERNAME", ""),
		SMTPPassword:       getenv("SMTP_PASSWORD", ""),
		SMTPFrom:           getenv("SMTP_FROM", "notifications@supremacy.local"),

		MapAssetsBaseURL: getenv("MAP_ASSETS_BASE_URL", ""),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeCoreAPI, ModeGateway, ModeEngine, ModeWorker, ModeAIBot:
	default:
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET must not be empty")
	}
	if c.Env == "prod" && strings.HasPrefix(c.JWTSecret, "dev-") {
		return errors.New("default dev JWT_SECRET is not allowed in prod")
	}
	if c.Env == "prod" && strings.HasPrefix(c.BotAPIKey, "dev-") {
		return errors.New("default dev BOT_API_KEY is not allowed in prod")
	}
	return nil
}

func defaultHTTPAddr(mode string) string {
	switch mode {
	case ModeCoreAPI:
		return ":8080"
	case ModeGateway:
		return ":8081"
	case ModeEngine:
		return ":8082"
	case ModeWorker:
		return ":8083"
	case ModeAIBot:
		return ":8084"
	default:
		return ":8080"
	}
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
