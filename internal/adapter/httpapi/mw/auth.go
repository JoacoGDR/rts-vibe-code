package mw

import (
	"context"
	"net/http"
	"strings"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/internal/auth"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

type ctxKey struct{ name string }

var claimsCtxKey = ctxKey{"claims"}

// RequireAuth extracts a Bearer token, verifies it via the issuer, and
// stashes the claims in the request context. Controllers retrieve them
// with [ClaimsFrom].
func RequireAuth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				render.Err(w, errs.New(errs.Unauthorized, "Authorization header required"))
				return
			}
			tok := strings.TrimPrefix(h, "Bearer ")
			claims, err := issuer.Verify(tok)
			if err != nil {
				render.Err(w, errs.Wrap(err, errs.Unauthorized, "bad token"))
				return
			}
			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFrom retrieves the JWT claims previously stashed by [RequireAuth]
// onto a request context. The bool is false when no claims are present
// (i.e. the route was not behind RequireAuth or the user is anonymous).
func ClaimsFrom(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsCtxKey).(*auth.Claims)
	return c, ok
}
