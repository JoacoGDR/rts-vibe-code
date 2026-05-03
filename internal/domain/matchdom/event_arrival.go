package matchdom

import "github.com/joaquing/clone-supremacy/internal/domain/timeline"

type arrivalHandler struct{}

func (arrivalHandler) Kind() timeline.Kind { return timeline.Arrival }

func (arrivalHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	u, ok := m.Units[ev.UnitID]
	if !ok || u.Version != ev.Version {
		return nil // stale event
	}
	prov, ok := m.Provinces[u.Dest]
	if !ok {
		return nil
	}
	m.Seq++
	u.OriginX = prov.X
	u.OriginY = prov.Y
	u.Origin = prov.ID
	u.StartedAt = ev.At
	u.ArrivesAt = ev.At
	out := []AppliedEvent{{
		Kind:     "arrival",
		OccurAt:  ev.At,
		Seq:      m.Seq,
		UnitID:   u.ID,
		Province: prov.ID,
		Slot:     u.OwnerSlot,
	}}
	out = append(out, m.resolveProvince(prov, ev.At)...)
	out = append(out, m.checkVictory(ev.At)...)
	return out
}
