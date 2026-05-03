package matchdom

import (
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
	"github.com/joaquing/clone-supremacy/internal/domain/unitdom"
)

type recruitCompleteHandler struct{}

func (recruitCompleteHandler) Kind() timeline.Kind { return timeline.RecruitComplete }

func (recruitCompleteHandler) Apply(m *Match, ev *timeline.Event) []AppliedEvent {
	var order *RecruitOrder
	var key string
	for k, o := range m.NextRecruit {
		if o.ID == ev.UnitID {
			order = o
			key = k
			break
		}
	}
	if order == nil {
		return nil
	}
	delete(m.NextRecruit, key)
	prov, ok := m.Provinces[order.Province]
	if !ok || prov.Owner != order.OwnerSlot {
		// Province changed hands while building — refund half the cost.
		if k, ok := unitdom.ByID(order.UnitType); ok {
			refund := k.Cost().Halved()
			if bank := m.Resources[order.OwnerSlot]; bank != nil {
				bank.Add(refund)
			}
		}
		return nil
	}
	kind, ok := unitdom.ByID(order.UnitType)
	if !ok {
		return nil
	}
	u := &Unit{
		ID:        uuid.New().String(),
		OwnerSlot: order.OwnerSlot,
		Type:      order.UnitType,
		HP:        kind.StartingHP(),
		Origin:    prov.ID,
		Dest:      prov.ID,
		OriginX:   prov.X,
		OriginY:   prov.Y,
		DestX:     prov.X,
		DestY:     prov.Y,
		Speed:     kind.Speed(),
		StartedAt: ev.At,
		ArrivesAt: ev.At,
		Version:   1,
	}
	m.Units[u.ID] = u
	m.Seq++
	return []AppliedEvent{{
		Kind: "unit_recruited", OccurAt: ev.At, Seq: m.Seq,
		UnitID: u.ID, Province: prov.ID, Slot: order.OwnerSlot,
		Extra: map[string]any{"type": order.UnitType},
	}}
}
