package errs_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/joaquing/clone-supremacy/pkg/errs"
)

func TestAsExtractsErr(t *testing.T) {
	original := errs.New(errs.NotFound, "user 42")
	wrapped := errs.Wrap(original, errs.Conflict, "race")
	got := errs.As(wrapped)
	if got == nil || got.Code != errs.Conflict {
		t.Fatalf("expected outer Conflict, got %v", got)
	}
	// Walking the chain still reaches the original.
	var inner *errs.Err
	if !errors.As(wrapped.Unwrap(), &inner) || inner.Code != errs.NotFound {
		t.Fatalf("expected wrapped NotFound, got %v", inner)
	}
}

func TestHTTPStatusMapping(t *testing.T) {
	cases := map[errs.Code]int{
		errs.BadRequest:      http.StatusBadRequest,
		errs.Unauthorized:    http.StatusUnauthorized,
		errs.Forbidden:       http.StatusForbidden,
		errs.NotFound:        http.StatusNotFound,
		errs.Conflict:        http.StatusConflict,
		errs.RateLimited:     http.StatusTooManyRequests,
		errs.UnavailableCode: http.StatusServiceUnavailable,
		errs.Internal:        http.StatusInternalServerError,
	}
	for code, want := range cases {
		if got := code.HTTPStatus(); got != want {
			t.Errorf("%s -> %d, want %d", code, got, want)
		}
	}
}

func TestWrapNilStaysNil(t *testing.T) {
	if errs.Wrap(nil, errs.Internal, "boom") != nil {
		t.Fatal("Wrap(nil) should return nil")
	}
}
