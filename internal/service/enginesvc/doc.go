// Package enginesvc orchestrates the simulation: it owns the Runner that
// drives one goroutine per hosted match, dispatches commands into them,
// advances game time, fans events out and produces per-slot state
// snapshots. It depends on domain (matchdom, visibility, timeline) and
// talks to the message bus via a Broadcaster port whose only impl today is
// the NATS bridge in internal/adapter/natsbridge.
package enginesvc
