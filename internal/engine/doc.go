// Package engine is the composition root for the engine binary mode.
// All simulation logic lives under internal/domain (matchdom, timeline,
// visibility, …) and internal/service/enginesvc; this package wires
// those parts up to NATS, Redis and the HTTP health surface.
package engine
