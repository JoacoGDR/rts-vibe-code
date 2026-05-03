package health

import (
	"encoding/json"
	"net/http"
)

// Check is the unit of work for a probe. Returning a non-nil error fails the
// probe; the error message is included in the response body.
type Check func() error

// Handler exposes /healthz (liveness) and /readyz (readiness). Each set of
// checks is evaluated on every request — checks should be cheap.
type Handler struct {
	mode    string
	version string
	live    []Check
	ready   []Check
}

func New(mode, version string) *Handler {
	return &Handler{mode: mode, version: version}
}

func (h *Handler) RegisterLive(c Check)  { h.live = append(h.live, c) }
func (h *Handler) RegisterReady(c Check) { h.ready = append(h.ready, c) }

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request)  { h.write(w, h.live) }
func (h *Handler) Ready(w http.ResponseWriter, _ *http.Request) { h.write(w, h.ready) }

func (h *Handler) write(w http.ResponseWriter, checks []Check) {
	for _, c := range checks {
		if err := c(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "fail",
				"error":  err.Error(),
				"mode":   h.mode,
			})
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"mode":    h.mode,
		"version": h.version,
	})
}
