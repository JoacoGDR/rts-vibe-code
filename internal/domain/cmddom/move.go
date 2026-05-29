package cmddom

import (
	"errors"
	"strconv"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/pathdom"
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
	if u.Speed <= 0 {
		return "no_speed", nil, errors.New("unit has no speed")
	}

	now := m.GameNow
	target, err := parseMoveTarget(m, cmd)
	if err != nil {
		return "bad_target", nil, err
	}

	queue := false
	if q, ok := cmd.Extra("queue"); ok {
		queue, _ = strconv.ParseBool(q)
	}

	if queue {
		u.Waypoints = append(u.Waypoints, target)
		if u.IsMoving(now) {
			return "ok", nil, nil
		}
		u.Version++
		if m.StartNextWaypoint(u, now) {
			return "ok", nil, nil
		}
		return "blocked_by_treaty", nil, ErrInvalidDest
	}

	g := pathdom.NewGraph(m.Map)
	curX, curY := u.PositionAt(now)
	legs, err := g.RouteFromEdge(curX, curY, u.Origin, u.Dest, target)
	if err != nil {
		return "no_route", nil, ErrInvalidDest
	}
	owners := m.ProvinceOwners()
	if pathdom.RouteBlocked(m.Diplomacy, string(cmd.IssuerSlot), owners, legs) {
		return "blocked_by_treaty", nil, ErrInvalidDest
	}

	u.Waypoints = nil
	u.Version++
	m.SetUnitPath(u, legs, now)
	return "ok", nil, nil
}

func parseMoveTarget(m *matchdom.Match, cmd Command) (pathdom.Target, error) {
	if kind, ok := cmd.Extra("target_kind"); ok {
		x, errX := strconv.ParseFloat(cmd.Args["target_x"], 64)
		y, errY := strconv.ParseFloat(cmd.Args["target_y"], 64)
		if errX != nil || errY != nil {
			return pathdom.Target{}, ErrInvalidDest
		}
		switch kind {
		case string(pathdom.TargetNode):
			prov := cmd.Args["target_province"]
			if prov == "" {
				prov = nearestProvince(m, x, y)
			}
			return pathdom.Target{Kind: pathdom.TargetNode, Province: prov, X: x, Y: y}, nil
		case string(pathdom.TargetEdge):
			t, _ := strconv.ParseFloat(cmd.Args["edge_t"], 64)
			return pathdom.Target{
				Kind: pathdom.TargetEdge,
				EdgeFrom: cmd.Args["edge_from"], EdgeTo: cmd.Args["edge_to"],
				X: x, Y: y, T: t,
			}, nil
		default:
			return pathdom.Target{}, ErrInvalidDest
		}
	}

	// Legacy province-to-province move.
	to, ok := m.Provinces[string(cmd.To)]
	if !ok {
		return pathdom.Target{}, ErrInvalidDest
	}
	if string(cmd.From) != "" {
		from, okFrom := m.Provinces[string(cmd.From)]
		if !okFrom {
			return pathdom.Target{}, ErrInvalidDest
		}
		if !m.Map.HasEdge(from.ID, to.ID) && from.ID != to.ID {
			return pathdom.Target{}, ErrInvalidDest
		}
	}
	return pathdom.Target{Kind: pathdom.TargetNode, Province: to.ID, X: to.X, Y: to.Y}, nil
}

func nearestProvince(m *matchdom.Match, x, y float64) string {
	g := pathdom.NewGraph(m.Map)
	t := g.SnapTarget(x, y)
	if t.Kind == pathdom.TargetNode {
		return t.Province
	}
	return t.EdgeFrom
}
