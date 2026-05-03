package cmddom

import "github.com/joaquing/clone-supremacy/internal/domain/matchdom"

// Handler is the strategy interface for every player command kind. Apply
// returns a short metric label, the list of [matchdom.AppliedEvent]s
// produced by the command (often nil — most commands enqueue future
// events on the timeline instead of emitting now), and an error.
//
// Commands that mutate the simulation in a way that has no observable
// instantaneous effect (e.g. recruit only schedules a future
// RecruitComplete) return an empty slice. Commands like the Phase 4
// diplomacy actions, which produce a single synchronous notification,
// return one entry.
type Handler interface {
	Kind() string
	Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error)
}
