package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is the canonical error category. Add new codes sparingly; if a code
// only exists to serve one call site it does not belong here.
type Code string

const (
	BadRequest      Code = "bad_request"
	Unauthorized    Code = "unauthorized"
	Forbidden       Code = "forbidden"
	NotFound        Code = "not_found"
	Conflict        Code = "conflict"
	RateLimited     Code = "rate_limited"
	UnavailableCode Code = "unavailable"
	Internal        Code = "internal"
)

// HTTPStatus maps a Code to its canonical HTTP status. Codes that don't have
// a 1:1 mapping default to 500.
func (c Code) HTTPStatus() int {
	switch c {
	case BadRequest:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	case RateLimited:
		return http.StatusTooManyRequests
	case UnavailableCode:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Err is the canonical error envelope. Always returned by value-pointer so
// errors.As can detect it.
type Err struct {
	Code  Code
	Msg   string
	Cause error
}

// New constructs a fresh error with no wrapped cause.
func New(code Code, msg string) *Err {
	return &Err{Code: code, Msg: msg}
}

// Wrap annotates an underlying error with a code + message. Returns nil when
// err is nil so callers can chain it without an extra check.
func Wrap(err error, code Code, msg string) *Err {
	if err == nil {
		return nil
	}
	return &Err{Code: code, Msg: msg, Cause: err}
}

func (e *Err) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.Cause)
}

// Unwrap exposes the cause for errors.Is / errors.As traversal.
func (e *Err) Unwrap() error { return e.Cause }

// HTTPStatus convenience for the renderer.
func (e *Err) HTTPStatus() int {
	if e == nil {
		return http.StatusOK
	}
	return e.Code.HTTPStatus()
}

// As returns the *Err embedded in err (or nil) using errors.As semantics.
// Saves callers from re-typing the same five lines.
func As(err error) *Err {
	var target *Err
	if errors.As(err, &target) {
		return target
	}
	return nil
}
