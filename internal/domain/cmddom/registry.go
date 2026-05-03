package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/ids"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

var registry = map[string]Handler{}

// Register adds a handler to the dispatcher. Last registration wins so
// tests can swap behaviour. Called from each handler's init().
func Register(h Handler) { registry[h.Kind()] = h }

func init() {
	Register(moveHandler{})
	Register(recruitHandler{})
	Register(constructHandler{})
	Register(declareWarHandler{})
	Register(proposePeaceHandler{})
	Register(proposeAllianceHandler{})
	Register(acceptPeaceHandler{})
	Register(acceptAllianceHandler{})
	Register(shareMapHandler{})
	Register(revokeShareMapHandler{})
	Register(rightOfWayGrantHandler{})
	Register(rightOfWayRevokeHandler{})
}

// Dispatch is the single entry point the runner calls. It validates the
// match status, resolves the issuer slot, and forwards to the matching
// handler. The returned events are produced synchronously by the command
// (mostly empty); the runner is responsible for then draining the
// timeline for any future events the command scheduled.
func Dispatch(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	if m.Status != "active" {
		return "match_inactive", nil, ErrMatchInactive
	}
	if cmd.IssuerSlot == "" && cmd.UserID != "" {
		for slot, p := range m.Players {
			if p.UserID == string(cmd.UserID) {
				cmd.IssuerSlot = ids.SlotID(slot)
				break
			}
		}
	}
	if cmd.IssuerSlot == "" {
		return "no_slot", nil, ErrUnauthorized
	}
	h, ok := registry[cmd.Kind]
	if !ok {
		return "unknown_kind", nil, ErrUnknownCommand
	}
	return h.Apply(m, cmd)
}
