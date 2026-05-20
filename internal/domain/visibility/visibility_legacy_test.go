package visibility

import (
	"math"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

// forLegacy is the pre-optimisation brute-force implementation kept for
// equivalence tests.
func forLegacy(m *matchdom.Match, slot string, at time.Time) (visibleProvinces map[string]bool, visibleUnits map[string]bool) {
	visibleProvinces = map[string]bool{}
	visibleUnits = map[string]bool{}

	type circle struct {
		x, y, r float64
	}
	circles := []circle{}
	contributors := computeViewerContributors(m, slot)

	for _, p := range m.Provinces {
		if !contributors[p.Owner] {
			continue
		}
		visibleProvinces[p.ID] = true
		for _, n := range m.Map.Neighbors(p.ID) {
			visibleProvinces[n] = true
		}
		circles = append(circles, circle{p.X, p.Y, balance.OwnedProvinceViewRadius})
	}

	for _, u := range m.Units {
		if !contributors[u.OwnerSlot] {
			continue
		}
		x, y := u.PositionAt(at)
		visibleUnits[u.ID] = true
		circles = append(circles, circle{x, y, balance.UnitViewRadius(u.Type)})
	}

	for _, p := range m.Provinces {
		if visibleProvinces[p.ID] {
			continue
		}
		for _, c := range circles {
			dx := p.X - c.x
			dy := p.Y - c.y
			if math.Sqrt(dx*dx+dy*dy) <= c.r {
				visibleProvinces[p.ID] = true
				break
			}
		}
	}

	for _, u := range m.Units {
		if visibleUnits[u.ID] {
			continue
		}
		ux, uy := u.PositionAt(at)
		for _, c := range circles {
			dx := ux - c.x
			dy := uy - c.y
			if math.Sqrt(dx*dx+dy*dy) <= c.r {
				visibleUnits[u.ID] = true
				break
			}
		}
	}
	return visibleProvinces, visibleUnits
}
