package render

import (
	"encoding/json"
	"net/http"

	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// JSON writes a successful response with the given status. Errors during
// encoding are intentionally swallowed because at that point we have already
// committed the status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Err walks the cause chain looking for an *errs.Err and renders it. If no
// such error is found the response is a generic 500. The wire shape is:
//
//	{"error": {"code": "...", "message": "..."}}
func Err(w http.ResponseWriter, err error) {
	var (
		status = http.StatusInternalServerError
		code   = errs.Internal
		msg    = "internal error"
	)
	if e := errs.As(err); e != nil {
		status = e.HTTPStatus()
		code = e.Code
		msg = e.Msg
	} else if err != nil {
		msg = err.Error()
	}
	JSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    string(code),
			"message": msg,
		},
	})
}
