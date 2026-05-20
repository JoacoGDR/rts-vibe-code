package visibility

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// For returns the set of province IDs and unit IDs that the given slot can
// perceive at the given game time. Computed by:
//
//  1. Every owned province is always visible to its owner, plus its
//     graph-adjacent neighbours.
//  2. Every owned unit illuminates a circle of `view_radius` around its
//     current interpolated position. Any province/unit whose position lies
//     inside the union of those circles is visible.
//  3. Alliance partners and ShareMap grantees contribute their circles to
//     the union.
//
// Circle–point tests use a uniform spatial index; contributor sets are
// cached on the match until diplomacy changes.
func For(m *matchdom.Match, slot string, at time.Time) (visibleProvinces map[string]bool, visibleUnits map[string]bool) {
	visibleProvinces = map[string]bool{}
	visibleUnits = map[string]bool{}

	contributors := viewerContributors(m, slot)

	circles := make([]visionCircle, 0, len(m.Provinces)+len(m.Units))

	for _, p := range m.Provinces {
		if !contributors[p.Owner] {
			continue
		}
		visibleProvinces[p.ID] = true
		for _, n := range m.Map.Neighbors(p.ID) {
			visibleProvinces[n] = true
		}
		r := balance.OwnedProvinceViewRadius
		circles = append(circles, visionCircle{p.X, p.Y, r * r})
	}

	for _, u := range m.Units {
		if !contributors[u.OwnerSlot] {
			continue
		}
		x, y := u.PositionAt(at)
		visibleUnits[u.ID] = true
		r := balance.UnitViewRadius(u.Type)
		circles = append(circles, visionCircle{x, y, r * r})
	}

	if len(circles) == 0 {
		return visibleProvinces, visibleUnits
	}

	points := make([][2]float64, 0, len(m.Provinces)+len(m.Units))
	for _, p := range m.Provinces {
		if visibleProvinces[p.ID] {
			continue
		}
		points = append(points, [2]float64{p.X, p.Y})
	}
	for _, u := range m.Units {
		if visibleUnits[u.ID] {
			continue
		}
		x, y := u.PositionAt(at)
		points = append(points, [2]float64{x, y})
	}
	idx := newSpatialIndex(circles, points)

	for _, p := range m.Provinces {
		if visibleProvinces[p.ID] {
			continue
		}
		if idx.within(p.X, p.Y, circles) {
			visibleProvinces[p.ID] = true
		}
	}

	for _, u := range m.Units {
		if visibleUnits[u.ID] {
			continue
		}
		ux, uy := u.PositionAt(at)
		if idx.within(ux, uy, circles) {
			visibleUnits[u.ID] = true
		}
	}
	return visibleProvinces, visibleUnits
}

// viewerContributors returns every slot whose vision is folded into
// `slot`'s perspective: itself, alliance partners (transitive coalition),
// and any slot that has granted `slot` a ShareMap pact. Results are cached
// on the match until diplomacy [diplomacydom.Registry.Version] changes.
func viewerContributors(m *matchdom.Match, slot string) map[string]bool {
	diplomacyVer := uint64(0)
	if m.Diplomacy != nil {
		diplomacyVer = m.Diplomacy.Version()
	}
	if m.VisContributors != nil && m.VisContributorVersion == diplomacyVer {
		if c, ok := m.VisContributors[slot]; ok {
			return c
		}
	} else {
		m.VisContributors = map[string]map[string]bool{}
		m.VisContributorVersion = diplomacyVer
	}

	out := computeViewerContributors(m, slot)
	m.VisContributors[slot] = out
	return out
}

func computeViewerContributors(m *matchdom.Match, slot string) map[string]bool {
	if m.Diplomacy == nil {
		return map[string]bool{slot: true}
	}
	allSlots := m.SlotIDs()
	out := m.Diplomacy.Coalition(slot, allSlots)
	for _, other := range allSlots {
		if out[other] {
			continue
		}
		if m.Diplomacy.SeesThrough(slot, other) {
			out[other] = true
		}
	}
	return out
}

// Filtered returns a slot-specific snapshot. Prefer [Batch.Filtered] when
// publishing state for many slots at the same instant.
func Filtered(m *matchdom.Match, slot string) *wire.MatchState {
	return NewBatch(m).Filtered(slot)
}

// Snapshot is the un-filtered "public" snapshot used by the persistence
// worker and (today) by the engine's general state subject. It includes
// everything; visibility filtering happens in [Filtered].
func Snapshot(m *matchdom.Match) *wire.MatchState {
	provinces := make([]wire.ProvinceState, 0, len(m.Provinces))
	for _, p := range m.Provinces {
		provinces = append(provinces, wire.ProvinceState{
			ID: p.ID, OwnerID: p.Owner,
			X: p.X, Y: p.Y, Capital: p.Capital,
		})
	}
	units := make([]wire.UnitState, 0, len(m.Units))
	for _, u := range m.Units {
		x, y := u.PositionAt(m.GameNow)
		units = append(units, matchdom.UnitToWire(u, x, y))
	}
	buildings := make([]wire.BuildingState, 0, len(m.Buildings))
	for _, b := range m.Buildings {
		buildings = append(buildings, wire.BuildingState{
			Type: b.Type, Province: b.Province, Owner: b.OwnerSlot,
		})
	}
	resources := map[string]wire.ResourcePool{}
	for slot, r := range m.Resources {
		resources[slot] = wire.ResourcePool{Manpower: r.Manpower, Food: r.Food, Iron: r.Iron}
	}
	return &wire.MatchState{
		MatchID:   m.ID,
		Tick:      m.Tick,
		GameTime:  m.GameNow,
		Provinces: provinces,
		Units:     units,
		Players:   playerStates(m),
		Resources: resources,
		Buildings: buildings,
		Queues:    QueueState(m, ""),
		Diplomacy: TreatyStates(m),
		Pacts:     PactsFor(m, ""),
	}
}

// QueueState extracts queued orders. If filterSlot is non-empty, only
// orders belonging to that slot are returned (so opponents can't read your
// production timing).
func QueueState(m *matchdom.Match, filterSlot string) wire.QueueState {
	var qs wire.QueueState
	for _, o := range m.NextRecruit {
		if filterSlot != "" && o.OwnerSlot != filterSlot {
			continue
		}
		qs.Recruits = append(qs.Recruits, wire.QueuedRecruit{
			Province: o.Province, UnitType: o.UnitType,
			Owner: o.OwnerSlot, CompletesAt: o.CompletesAt,
		})
	}
	for _, o := range m.NextBuild {
		if filterSlot != "" && o.OwnerSlot != filterSlot {
			continue
		}
		qs.Constructions = append(qs.Constructions, wire.QueuedConstruction{
			Province: o.Province, BuildingType: o.BuildingType,
			Owner: o.OwnerSlot, CompletesAt: o.CompletesAt,
		})
	}
	return qs
}
