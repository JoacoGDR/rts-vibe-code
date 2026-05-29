package matchdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/pathdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

const collisionRadius = 30

// PathLeg is one scheduled movement segment for a unit.
type PathLeg struct {
	FromProv, ToProv string
	FromX, FromY     float64
	ToX, ToY         float64
	ArrivesAt        time.Time
}

// ProvinceOwners builds a pathdom owner map from match provinces.
func (m *Match) ProvinceOwners() pathdom.ProvinceOwners {
	out := make(pathdom.ProvinceOwners, len(m.Provinces))
	for id, p := range m.Provinces {
		out[id] = p.Owner
	}
	return out
}

// SetUnitPath replaces the unit path from pathdom legs starting at now.
func (m *Match) SetUnitPath(u *Unit, legs []pathdom.Leg, now time.Time) {
	if len(legs) == 0 {
		return
	}
	scheduled := make([]PathLeg, 0, len(legs))
	cursor := now
	u.PathIndex = 0
	for _, l := range legs {
		dist := EuclidDistance(l.FromX, l.FromY, l.ToX, l.ToY)
		travel := time.Duration(dist/u.Speed) * time.Second
		if travel < time.Millisecond {
			travel = time.Millisecond
		}
		arrives := cursor.Add(travel)
		scheduled = append(scheduled, PathLeg{
			FromProv: l.FromProv, ToProv: l.ToProv,
			FromX: l.FromX, FromY: l.FromY, ToX: l.ToX, ToY: l.ToY,
			ArrivesAt: arrives,
		})
		cursor = arrives
	}
	u.Path = scheduled
	m.beginCurrentLeg(u, now)
	m.scheduleArrival(u)
}

func (m *Match) beginCurrentLeg(u *Unit, now time.Time) {
	if u.PathIndex >= len(u.Path) {
		return
	}
	leg := u.Path[u.PathIndex]
	u.Origin = leg.FromProv
	u.Dest = leg.ToProv
	u.OriginX, u.OriginY = leg.FromX, leg.FromY
	u.DestX, u.DestY = leg.ToX, leg.ToY
	if now.After(leg.ArrivesAt) {
		now = leg.ArrivesAt
	}
	u.StartedAt = now
	u.ArrivesAt = leg.ArrivesAt
}

func (m *Match) scheduleArrival(u *Unit) {
	if u.PathIndex >= len(u.Path) {
		return
	}
	m.Timeline.Push(&timeline.Event{
		At:      u.ArrivesAt,
		Kind:    timeline.Arrival,
		UnitID:  u.ID,
		Version: u.Version,
	})
}

// StartNextWaypoint routes to the next queued waypoint if the unit is idle.
func (m *Match) StartNextWaypoint(u *Unit, now time.Time) bool {
	if len(u.Waypoints) == 0 || u.IsMoving(now) {
		return false
	}
	wp := u.Waypoints[0]
	u.Waypoints = u.Waypoints[1:]
	g := pathdom.NewGraph(m.Map)
	legs, err := g.RouteFromEdge(u.OriginX, u.OriginY, u.Origin, u.Dest, wp)
	if err != nil {
		return false
	}
	owners := m.ProvinceOwners()
	if pathdom.RouteBlocked(m.Diplomacy, u.OwnerSlot, owners, legs) {
		return false
	}
	m.SetUnitPath(u, legs, now)
	return true
}

// CompleteUnitLeg finishes the current path leg and may chain or fight.
func (m *Match) CompleteUnitLeg(u *Unit, at time.Time) []AppliedEvent {
	if u.PathIndex >= len(u.Path) {
		return nil
	}
	leg := u.Path[u.PathIndex]
	u.OriginX, u.OriginY = leg.ToX, leg.ToY
	u.DestX, u.DestY = leg.ToX, leg.ToY
	u.Origin = leg.ToProv
	u.Dest = leg.ToProv
	u.StartedAt = at
	u.ArrivesAt = at

	m.Seq++
	out := []AppliedEvent{{
		Kind: "arrival", OccurAt: at, Seq: m.Seq,
		UnitID: u.ID, Province: leg.ToProv, Slot: u.OwnerSlot,
	}}

	owners := m.ProvinceOwners()
	pLeg := pathdom.Leg{FromProv: leg.FromProv, ToProv: leg.ToProv, FromX: leg.FromX, FromY: leg.FromY, ToX: leg.ToX, ToY: leg.ToY}
	if pathdom.HostileLegEnd(m.Diplomacy, u.OwnerSlot, owners, pLeg) {
		u.Path = nil
		u.PathIndex = 0
		u.Waypoints = nil
		if prov, ok := m.Provinces[leg.ToProv]; ok {
			out = append(out, m.resolveProvince(prov, at)...)
		}
		out = append(out, m.checkVictory(at)...)
		return out
	}

	if enemy := m.hostileUnitOnLeg(u, leg, at); enemy != nil {
		u.Path = nil
		u.PathIndex = 0
		u.Waypoints = nil
		if prov, ok := m.Provinces[leg.ToProv]; ok {
			out = append(out, m.resolveProvince(prov, at)...)
		}
		out = append(out, m.checkVictory(at)...)
		return out
	}

	u.PathIndex++
	if u.PathIndex < len(u.Path) {
		m.beginCurrentLeg(u, at)
		m.scheduleArrival(u)
		out = append(out, m.checkVictory(at)...)
		return out
	}

	u.Path = nil
	u.PathIndex = 0
	if m.StartNextWaypoint(u, at) {
		out = append(out, m.checkVictory(at)...)
		return out
	}
	if prov, ok := m.Provinces[leg.ToProv]; ok && !u.IsMoving(at) {
		out = append(out, m.resolveProvince(prov, at)...)
	}
	out = append(out, m.checkVictory(at)...)
	return out
}

func (m *Match) hostileUnitOnLeg(u *Unit, leg PathLeg, at time.Time) *Unit {
	for _, other := range m.Units {
		if other.ID == u.ID || other.HP <= 0 {
			continue
		}
		if m.Diplomacy.Stance(u.OwnerSlot, other.OwnerSlot) != diplomacydom.War {
			continue
		}
		if !sharesLegEdge(leg, other) {
			continue
		}
		ox, oy := other.PositionAt(at)
		if EuclidDistance(u.OriginX, u.OriginY, ox, oy) <= collisionRadius {
			return other
		}
	}
	return nil
}

func sharesLegEdge(leg PathLeg, other *Unit) bool {
	if len(other.Path) == 0 || other.PathIndex >= len(other.Path) {
		return other.Origin == leg.FromProv && other.Dest == leg.ToProv ||
			other.Origin == leg.ToProv && other.Dest == leg.FromProv
	}
	ol := other.Path[other.PathIndex]
	return (ol.FromProv == leg.FromProv && ol.ToProv == leg.ToProv) ||
		(ol.FromProv == leg.ToProv && ol.ToProv == leg.FromProv)
}
