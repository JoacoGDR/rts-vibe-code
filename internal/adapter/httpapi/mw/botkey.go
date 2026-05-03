package mw

import (
	"crypto/subtle"
	"net/http"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// BotKeyHeader is the HTTP header name the ai-bot subsystem uses to
// authenticate against the bot-only endpoints (login-as-bot, list bots).
// We deliberately use a custom header rather than Authorization so the
// existing JWT middleware does not try to interpret the value.
const BotKeyHeader = "X-Bot-API-Key"

// RequireBotKey is a middleware that rejects every request whose
// X-Bot-API-Key header does not match the pre-shared secret. The
// comparison uses subtle.ConstantTimeCompare so timing attacks cannot
// recover the key one byte at a time.
func RequireBotKey(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				render.Err(w, errs.New(errs.Forbidden, "bot endpoints disabled (BOT_API_KEY unset)"))
				return
			}
			provided := r.Header.Get(BotKeyHeader)
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				render.Err(w, errs.New(errs.Unauthorized, "invalid bot api key"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
