package matchdom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

type macroPulseHandler struct{}

func (macroPulseHandler) Kind() timeline.Kind { return timeline.MacroPulse }

func (macroPulseHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	out := []AppliedEvent{}
	for slot, bank := range m.Resources {
		rev := macroPulseRevenue(m, slot)
		bank.Add(rev)
		m.Seq++
		out = append(out, AppliedEvent{
			Kind: "macro_pulse", OccurAt: ev.At, Seq: m.Seq, Slot: slot,
			Extra: map[string]any{
				"manpower": bank.Manpower,
				"food":     bank.Food,
				"iron":     bank.Iron,
				"gain":     rev,
			},
		})
	}
	m.Timeline.Push(&timeline.Event{At: ev.At.Add(balance.MacroPulseInterval), Kind: timeline.MacroPulse})
	return out
}
