package matchdom

import "github.com/joaquing/clone-supremacy/internal/domain/timeline"

// EventHandler is the strategy interface for every timeline event kind.
// Apply mutates the match in response to the event and returns the
// AppliedEvents the runner should fan out.
type EventHandler interface {
	Kind() timeline.Kind
	Apply(m *Match, ev *timeline.Event) []AppliedEvent
}

var eventRegistry = map[timeline.Kind]EventHandler{}

// RegisterEventHandler adds a handler to the dispatcher. Last
// registration wins so tests can swap behaviour. Called from init().
func RegisterEventHandler(h EventHandler) { eventRegistry[h.Kind()] = h }

func init() {
	RegisterEventHandler(arrivalHandler{})
	RegisterEventHandler(macroPulseHandler{})
	RegisterEventHandler(recruitCompleteHandler{})
	RegisterEventHandler(constructCompleteHandler{})
	RegisterEventHandler(victoryCheckHandler{})
}
