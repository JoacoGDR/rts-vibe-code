package matchdom

import "github.com/joaquing/clone-supremacy/internal/domain/timeline"

type victoryCheckHandler struct{}

func (victoryCheckHandler) Kind() timeline.Kind { return timeline.VictoryCheck }

func (victoryCheckHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	return m.checkVictory(ev.At)
}
