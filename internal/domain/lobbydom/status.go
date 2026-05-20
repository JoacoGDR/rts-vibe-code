package lobbydom

// Status is the lobby lifecycle phase.
type Status string

const (
	StatusWaiting   Status = "waiting"
	StatusStarting  Status = "starting"
	StatusActive    Status = "active"
	StatusEnded     Status = "ended"
	StatusAbandoned Status = "abandoned"
)

// CanTransitionTo reports whether a match may move from s to next.
func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case StatusWaiting:
		return next == StatusStarting
	case StatusStarting:
		return next == StatusActive
	case StatusActive:
		return next == StatusEnded || next == StatusAbandoned
	case StatusEnded, StatusAbandoned:
		return false
	default:
		return false
	}
}
