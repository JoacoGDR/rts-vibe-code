package visibility

import (
	"math"
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
//  3. Phase 4: every alliance partner and every slot that has granted
//     `slot` a [diplomacydom.ShareMap] pact contributes its own circles
//     to the union, so allied vision stacks transparently.
//
// This is the engine-internal version of the "graph-based spatial
// partitioning" described in the architecture PDF — kept simple for the MVP
// (linear N×M scan) so we can tune correctness before optimising. Phase 8
// will lift this into its own service if profiling demands it.
func For(m *matchdom.Match, slot string, at time.Time) (visibleProvinces map[string]bool, visibleUnits map[string]bool) {
	visibleProvinces = map[string]bool{}
	visibleUnits = map[string]bool{}

	type circle struct {
		x, y, r float64
	}
	circles := []circle{}

	contributors := viewerContributors(m, slot)

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
			if dist(p.X, p.Y, c.x, c.y) <= c.r {
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
			if dist(ux, uy, c.x, c.y) <= c.r {
				visibleUnits[u.ID] = true
				break
			}
		}
	}
	return
}

// viewerContributors returns every slot whose vision is folded into
// `slot`'s perspective: itself, alliance partners (transitive coalition),
// and any slot that has granted `slot` a ShareMap pact.
func viewerContributors(m *matchdom.Match, slot string) map[string]bool {
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

func dist(ax, ay, bx, by float64) float64 {
	dx := ax - bx
	dy := ay - by
	return math.Sqrt(dx*dx + dy*dy)
}

// Filtered returns a slot-specific snapshot. Provinces and units the slot
// cannot see are stripped from the payload; resources are private to the
// requesting slot.
func Filtered(m *matchdom.Match, slot string) *wire.MatchState {
	provinces, units := For(m, slot, m.GameNow)

	pOut := make([]wire.ProvinceState, 0, len(provinces))
	for _, p := range m.Provinces {
		if !provinces[p.ID] {
			continue
		}
		pOut = append(pOut, wire.ProvinceState{
			ID: p.ID, OwnerID: p.Owner,
			X: p.X, Y: p.Y, Capital: p.Capital,
		})
	}
	uOut := make([]wire.UnitState, 0, len(units))
	for _, u := range m.Units {
		if !units[u.ID] {
			continue
		}
		x, y := u.PositionAt(m.GameNow)
		uOut = append(uOut, wire.UnitState{
			ID: u.ID, OwnerID: u.OwnerSlot, Type: u.Type,
			X: x, Y: y, HP: u.HP,
			Origin: u.Origin, Dest: u.Dest,
			StartedAt: u.StartedAt, ArrivesAt: u.ArrivesAt,
		})
	}
	players := make([]wire.PlayerState, 0, len(m.Players))
	for _, p := range m.Players {
		players = append(players, wire.PlayerState{
			ID: p.Slot, Name: p.Name, Color: p.Color, Alive: p.Alive,
		})
	}
	resources := map[string]wire.ResourcePool{}
	if r, ok := m.Resources[slot]; ok {
		resources[slot] = wire.ResourcePool{Manpower: r.Manpower, Food: r.Food, Iron: r.Iron}
	}
	buildings := []wire.BuildingState{}
	for _, b := range m.Buildings {
		if !provinces[b.Province] {
			continue
		}
		buildings = append(buildings, wire.BuildingState{
			Type: b.Type, Province: b.Province, Owner: b.OwnerSlot,
		})
	}
	queues := QueueState(m, slot)
	return &wire.MatchState{
		MatchID:   m.ID,
		Tick:      m.Tick,
		GameTime:  m.GameNow,
		Provinces: pOut,
		Units:     uOut,
		Players:   players,
		Resources: resources,
		Buildings: buildings,
		Queues:    queues,
		Diplomacy: TreatyStates(m),
		Pacts:     PactsFor(m, slot),
	}
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
		units = append(units, wire.UnitState{
			ID: u.ID, OwnerID: u.OwnerSlot, Type: u.Type,
			X: x, Y: y, HP: u.HP,
			Origin: u.Origin, Dest: u.Dest,
			StartedAt: u.StartedAt, ArrivesAt: u.ArrivesAt,
		})
	}
	players := make([]wire.PlayerState, 0, len(m.Players))
	for _, p := range m.Players {
		players = append(players, wire.PlayerState{
			ID: p.Slot, Name: p.Name, Color: p.Color, Alive: p.Alive,
		})
	}
	resources := map[string]wire.ResourcePool{}
	for slot, r := range m.Resources {
		resources[slot] = wire.ResourcePool{Manpower: r.Manpower, Food: r.Food, Iron: r.Iron}
	}
	buildings := make([]wire.BuildingState, 0, len(m.Buildings))
	for _, b := range m.Buildings {
		buildings = append(buildings, wire.BuildingState{
			Type: b.Type, Province: b.Province, Owner: b.OwnerSlot,
		})
	}
	queues := QueueState(m, "")
	return &wire.MatchState{
		MatchID:   m.ID,
		Tick:      m.Tick,
		GameTime:  m.GameNow,
		Provinces: provinces,
		Units:     units,
		Players:   players,
		Resources: resources,
		Buildings: buildings,
		Queues:    queues,
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
