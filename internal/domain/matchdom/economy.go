package matchdom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/buildingdom"
	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

// macroPulseRevenue returns the resource trickle a slot earns per pulse,
// taking province ownership and (Phase 4+) buildings into account. Per-
// province income comes from balance.ProvinceRevenue; per-building bonus
// comes from each registered building kind's Production().
func macroPulseRevenue(m *Match, slot string) economydom.Resources {
	out := economydom.Resources{}
	provinceShare := balance.ProvinceRevenue()
	for _, p := range m.Provinces {
		if p.Owner != slot {
			continue
		}
		out.Add(provinceShare)
	}
	for _, b := range m.Buildings {
		if b.OwnerSlot != slot {
			continue
		}
		if kind, ok := buildingdom.ByID(b.Type); ok {
			out.Add(kind.Production())
		}
	}
	return out
}
