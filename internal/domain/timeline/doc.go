// Package timeline provides the engine's discrete-event scheduler. Events
// (arrival, combat tick, macro pulse, victory check, recruit/build complete)
// live on a min-heap keyed by game time so the simulation only does work
// when something is actually due.
package timeline
