package natsbridge

import "fmt"

// Stream/subject constants. The wildcard subjects are used for stream and
// consumer configuration; per-match subjects are built with the helpers
// below.
const (
	StreamName    = "supremacy-cmd"
	SubjCmdAll    = "match.*.cmd"
	SubjStateAll  = "match.*.state"
	SubjEventAll  = "match.*.event"
	SubjStartAll  = "match.*.start"
	SubjResyncAll = "match.*.resync"
)

// CmdSubject returns the subject the gateway publishes player commands on.
func CmdSubject(matchID string) string { return fmt.Sprintf("match.%s.cmd", matchID) }

// StartSubject returns the subject core-api uses to ask the engine to host
// a new match.
func StartSubject(matchID string) string { return fmt.Sprintf("match.%s.start", matchID) }

// ResyncSubject is used by the gateway to ask the engine to rebroadcast.
func ResyncSubject(matchID string) string { return fmt.Sprintf("match.%s.resync", matchID) }

// PublicStateSubject is the un-filtered state stream (used by the
// persistence worker and lobby observers).
func PublicStateSubject(matchID string) string { return fmt.Sprintf("match.%s.state", matchID) }

// PublicEventSubject is the un-filtered event stream.
func PublicEventSubject(matchID string) string { return fmt.Sprintf("match.%s.event", matchID) }

// SlotStateSubject is the per-slot filtered state subject.
func SlotStateSubject(matchID, slot string) string {
	return fmt.Sprintf("match.%s.slot.%s.state", matchID, slot)
}

// SlotEventSubject is the per-slot filtered event subject.
func SlotEventSubject(matchID, slot string) string {
	return fmt.Sprintf("match.%s.slot.%s.event", matchID, slot)
}

// FinalStateSubject is the one-shot snapshot published when a match
// finishes its post-game cleanup. The worker subscribes here to mark
// the lobby row ended and persist the final snapshot.
func FinalStateSubject(matchID string) string {
	return fmt.Sprintf("match.%s.state.final", matchID)
}

// EndMatchSubject asks the engine to tear down an in-memory match
// (abandonment path). Payload is [EndMatchPayload].
func EndMatchSubject(matchID string) string {
	return fmt.Sprintf("match.%s.end", matchID)
}
