package timeline

import (
	"container/heap"
	"time"
)

// Kind enumerates the events the engine schedules on its timeline.
type Kind string

const (
	Arrival           Kind = "arrival"
	CombatStart       Kind = "combat_start"
	CombatTick        Kind = "combat_tick"
	MacroPulse        Kind = "macro_pulse"
	VictoryCheck      Kind = "victory_check"
	RecruitComplete   Kind = "recruit_complete"
	ConstructComplete Kind = "construct_complete"
	MatchCleanup      Kind = "match_cleanup"
)

// Event is the unit of work the scheduler holds. The Version field
// implements the "validation signature" trick from the architecture doc:
// when an event pops, callers only act on it if the unit's command version
// still matches.
type Event struct {
	At      time.Time
	Kind    Kind
	UnitID  string
	Other   string // optional secondary unit id for combat
	Version uint64
	index   int
}

type heapImpl []*Event

func (h heapImpl) Len() int           { return len(h) }
func (h heapImpl) Less(i, j int) bool { return h[i].At.Before(h[j].At) }
func (h heapImpl) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *heapImpl) Push(x any) {
	ev := x.(*Event)
	ev.index = len(*h)
	*h = append(*h, ev)
}
func (h *heapImpl) Pop() any {
	old := *h
	n := len(old)
	ev := old[n-1]
	old[n-1] = nil
	ev.index = -1
	*h = old[:n-1]
	return ev
}

// Timeline is a thin priority queue facade over a min-heap.
type Timeline struct{ h heapImpl }

// New returns an empty timeline.
func New() *Timeline { t := &Timeline{}; heap.Init(&t.h); return t }

// Push schedules a new event.
func (t *Timeline) Push(e *Event) { heap.Push(&t.h, e) }

// Peek returns the next event without removing it.
func (t *Timeline) Peek() *Event {
	if len(t.h) == 0 {
		return nil
	}
	return t.h[0]
}

// Pop removes and returns the next event, or nil if empty.
func (t *Timeline) Pop() *Event {
	if len(t.h) == 0 {
		return nil
	}
	return heap.Pop(&t.h).(*Event)
}

// Len returns the number of pending events.
func (t *Timeline) Len() int { return t.h.Len() }
