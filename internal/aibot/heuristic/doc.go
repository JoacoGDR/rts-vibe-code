// Package heuristic owns the decision-making code the ai-bot mode runs
// against every state envelope it receives. The heuristic is pure Go and
// has no I/O dependencies so it can be unit-tested in isolation against
// fixture states. The bot session in `internal/aibot/session.go` is
// responsible for translating the heuristic's `wire.Command` outputs back
// onto the WebSocket transport.
//
// The current strategy is intentionally simple — keep capital garrisoned,
// recruit when affordable, send idle units toward the closest hostile
// province, never attack peace/alliance, accept peace when the
// capital-count is at risk, and declare war on the perceived leader
// (most provinces). Future iterations can add unit-type preferences,
// research, multi-step pathing, etc.
package heuristic
