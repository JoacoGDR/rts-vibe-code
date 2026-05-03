package matchdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

// AppliedEvent is what the engine emits after handling a timeline event
// or applying a command. The runner serialises these to NATS so the
// gateway can fan them out to clients.
type AppliedEvent struct {
	Kind     string         `json:"kind"`
	OccurAt  time.Time      `json:"occur_at"`
	Seq      uint64         `json:"seq"`
	UnitID   string         `json:"unit_id,omitempty"`
	Province string         `json:"province,omitempty"`
	Slot     string         `json:"slot,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// ProcessNext drains the next ready timeline events and returns the
// AppliedEvents they produced. Stale events (validation signature
// mismatch) are dropped silently.
func (m *Match) ProcessNext(now time.Time) []AppliedEvent {
	out := []AppliedEvent{}
	for {
		next := m.Timeline.Peek()
		if next == nil || next.At.After(now) {
			return out
		}
		ev := m.Timeline.Pop()
		applied := m.HandleEvent(ev)
		out = append(out, applied...)
	}
}

// HandleEvent dispatches one timeline event to its registered handler.
// Exposed so tests can poke individual events; the runner uses
// ProcessNext.
func (m *Match) HandleEvent(ev *timeline.Event) []AppliedEvent {
	h, ok := eventRegistry[ev.Kind]
	if !ok {
		return nil
	}
	return h.Apply(m, ev)
}
