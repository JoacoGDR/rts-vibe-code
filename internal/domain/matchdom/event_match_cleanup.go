package matchdom

import "github.com/joaquing/clone-supremacy/internal/domain/timeline"

type matchCleanupHandler struct{}

func init() { RegisterEventHandler(matchCleanupHandler{}) }

func (matchCleanupHandler) Kind() timeline.Kind { return timeline.MatchCleanup }

func (matchCleanupHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	if m.CleanupDone {
		return nil
	}
	m.CleanupDone = true
	m.Seq++
	return []AppliedEvent{{
		Kind: "match_cleanup", OccurAt: ev.At, Seq: m.Seq,
		Extra: map[string]any{"winner": m.WinnerSlot, "winners": m.WinnerCoal},
	}}
}
