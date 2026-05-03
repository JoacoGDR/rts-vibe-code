package cmddom

import (
	"errors"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

type moveHandler struct{}

func (moveHandler) Kind() string { return "move" }

func (moveHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	u, ok := m.Units[string(cmd.UnitID)]
	if !ok {
		return "unknown_unit", nil, ErrUnknownUnit
	}
	if u.OwnerSlot != string(cmd.IssuerSlot) {
		return "unauthorized", nil, ErrUnauthorized
	}
	from, ok := m.Provinces[string(cmd.From)]
	if !ok {
		return "bad_origin", nil, ErrInvalidDest
	}
	to, ok := m.Provinces[string(cmd.To)]
	if !ok {
		return "bad_dest", nil, ErrInvalidDest
	}
	if !m.Map.HasEdge(from.ID, to.ID) && from.ID != to.ID {
		return "no_edge", nil, ErrInvalidDest
	}
	if u.Speed <= 0 {
		return "no_speed", nil, errors.New("unit has no speed")
	}
	if !m.Diplomacy.MayMoveThrough(string(cmd.IssuerSlot), to.Owner) &&
		m.Diplomacy.Stance(string(cmd.IssuerSlot), to.Owner) != diplomacydom.War {
		return "blocked_by_treaty", nil, ErrInvalidDest
	}

	now := m.GameNow
	curX, curY := u.PositionAt(now)
	dist := matchdom.EuclidDistance(curX, curY, to.X, to.Y)
	travel := time.Duration(dist/u.Speed) * time.Second

	u.Origin = from.ID
	u.Dest = to.ID
	u.OriginX, u.OriginY = curX, curY
	u.DestX, u.DestY = to.X, to.Y
	u.StartedAt = now
	u.ArrivesAt = now.Add(travel)
	u.Version++

	m.Timeline.Push(&timeline.Event{
		At:      u.ArrivesAt,
		Kind:    timeline.Arrival,
		UnitID:  u.ID,
		Version: u.Version,
	})
	return "ok", nil, nil
}
