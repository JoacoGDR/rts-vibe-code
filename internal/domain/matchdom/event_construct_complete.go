package matchdom

import "github.com/joaquing/clone-supremacy/internal/domain/timeline"

type constructCompleteHandler struct{}

func (constructCompleteHandler) Kind() timeline.Kind { return timeline.ConstructComplete }

func (constructCompleteHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	var order *BuildOrder
	var key string
	for k, o := range m.NextBuild {
		if o.ID == ev.UnitID {
			order = o
			key = k
			break
		}
	}
	if order == nil {
		return nil
	}
	delete(m.NextBuild, key)
	prov, ok := m.Provinces[order.Province]
	if !ok || prov.Owner != order.OwnerSlot {
		return nil
	}
	m.Buildings = append(m.Buildings, &Building{
		Type: order.BuildingType, Province: prov.ID, OwnerSlot: order.OwnerSlot,
	})
	m.Seq++
	return []AppliedEvent{{
		Kind: "building_constructed", OccurAt: ev.At, Seq: m.Seq,
		Province: prov.ID, Slot: order.OwnerSlot,
		Extra: map[string]any{"type": order.BuildingType},
	}}
}
