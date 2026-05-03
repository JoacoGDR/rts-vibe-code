package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// New constructs the root logger. Level can be one of debug/info/warn/error
// (case-insensitive); format is "json" (default) or "text".
func New(level, format, mode, env string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl, AddSource: lvl == slog.LevelDebug}

	var h slog.Handler
	if strings.ToLower(format) == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(h).With(
		slog.String("service", "supremacy"),
		slog.String("mode", mode),
		slog.String("env", env),
	)
	slog.SetDefault(logger)
	return logger
}

// Inject stores the logger on the context so handlers further down can pull
// a request-scoped logger via [From].
func Inject(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From retrieves the logger off the context. Falls back to the default if
// none was injected so callers never have to nil-check.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
