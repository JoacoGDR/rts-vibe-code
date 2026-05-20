package visibility

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// visibleSet is the output of [For] for one slot at one instant.
type visibleSet struct {
	Provinces map[string]bool
	Units     map[string]bool
}

// Batch amortises visibility and snapshot work across many slots and
// events in one engine publish cycle. Construct with [NewBatch] at the
// start of [enginesvc.Runner.broadcastEvents] (or any path that fans out
// state/events for the same [matchdom.Match.GameNow]).
type Batch struct {
	m   *matchdom.Match
	at  time.Time
	vis map[string]visibleSet

	players  []wire.PlayerState
	treaties []wire.TreatyState
}

// NewBatch captures the match and game instant used for visibility.
func NewBatch(m *matchdom.Match) *Batch {
	return &Batch{
		m:  m,
		at: m.GameNow,
		vis: make(map[string]visibleSet),
	}
}

// At returns the game time used for this batch.
func (b *Batch) At() time.Time { return b.at }

// Visible returns province and unit visibility for slot, computing [For]
// at most once per slot per batch.
func (b *Batch) Visible(slot string) (provinces, units map[string]bool) {
	if v, ok := b.vis[slot]; ok {
		return v.Provinces, v.Units
	}
	p, u := For(b.m, slot, b.at)
	b.vis[slot] = visibleSet{Provinces: p, Units: u}
	return p, u
}

// SlotObservesEvent is the batch-aware version of engine event fan-out.
func (b *Batch) SlotObservesEvent(slot string, e matchdom.AppliedEvent) bool {
	provinces, units := b.Visible(slot)
	if e.Slot == slot {
		return true
	}
	if e.UnitID != "" && units[e.UnitID] {
		return true
	}
	if e.Province != "" && provinces[e.Province] {
		return true
	}
	switch e.Kind {
	case "match_ended", "province_captured":
		return true
	}
	return false
}

// ensureShared builds player and treaty wire slices once per batch.
func (b *Batch) ensureShared() {
	if b.players != nil {
		return
	}
	b.players = playerStates(b.m)
	b.treaties = TreatyStates(b.m)
}

// Filtered returns a slot-specific snapshot using cached visibility and
// shared invariant slices.
func (b *Batch) Filtered(slot string) *wire.MatchState {
	provinces, units := b.Visible(slot)
	b.ensureShared()

	pOut := make([]wire.ProvinceState, 0, len(provinces))
	for _, p := range b.m.Provinces {
		if !provinces[p.ID] {
			continue
		}
		pOut = append(pOut, wire.ProvinceState{
			ID: p.ID, OwnerID: p.Owner,
			X: p.X, Y: p.Y, Capital: p.Capital,
		})
	}
	uOut := make([]wire.UnitState, 0, len(units))
	for _, u := range b.m.Units {
		if !units[u.ID] {
			continue
		}
		x, y := u.PositionAt(b.at)
		uOut = append(uOut, matchdom.UnitToWire(u, x, y))
	}
	resources := map[string]wire.ResourcePool{}
	if r, ok := b.m.Resources[slot]; ok {
		resources[slot] = wire.ResourcePool{Manpower: r.Manpower, Food: r.Food, Iron: r.Iron}
	}
	buildings := make([]wire.BuildingState, 0, len(b.m.Buildings))
	for _, bl := range b.m.Buildings {
		if !provinces[bl.Province] {
			continue
		}
		buildings = append(buildings, wire.BuildingState{
			Type: bl.Type, Province: bl.Province, Owner: bl.OwnerSlot,
		})
	}
	return &wire.MatchState{
		MatchID:   b.m.ID,
		Tick:      b.m.Tick,
		GameTime:  b.at,
		Provinces: pOut,
		Units:     uOut,
		Players:   b.players,
		Resources: resources,
		Buildings: buildings,
		Queues:    QueueState(b.m, slot),
		Diplomacy: b.treaties,
		Pacts:     PactsFor(b.m, slot),
	}
}

func playerStates(m *matchdom.Match) []wire.PlayerState {
	out := make([]wire.PlayerState, 0, len(m.Players))
	for _, p := range m.Players {
		out = append(out, wire.PlayerState{
			ID: p.Slot, Name: p.Name, Color: p.Color, Alive: p.Alive,
		})
	}
	return out
}
