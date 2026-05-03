package cmddom

import (
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
	"github.com/joaquing/clone-supremacy/internal/domain/unitdom"
)

type recruitHandler struct{}

func (recruitHandler) Kind() string { return "recruit" }

func (recruitHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	prov, ok := m.Provinces[string(cmd.From)]
	if !ok {
		return "bad_province", nil, ErrInvalidProvince
	}
	if prov.Owner != string(cmd.IssuerSlot) {
		return "unauthorized", nil, ErrUnauthorized
	}
	unitType := string(cmd.UnitID)
	if t, ok := cmd.Extra("type"); ok {
		unitType = t
	}
	kind, ok := unitdom.ByID(unitType)
	if !ok {
		return "bad_unit_type", nil, ErrUnknownUnitType
	}
	if req := kind.NeedsBuilding(); req != "" && !matchdom.HasBuilding(m, prov.ID, req) {
		return "needs_factory", nil, ErrFactoryRequired
	}
	if _, busy := m.NextRecruit[prov.ID]; busy {
		return "busy", nil, ErrAlreadyQueued
	}

	bank := m.Resources[string(cmd.IssuerSlot)]
	cost := kind.Cost()
	if bank == nil || !bank.CanPay(cost) {
		return "no_funds", nil, ErrInsufficientFunds
	}
	bank.Pay(cost)

	order := &matchdom.RecruitOrder{
		ID:          uuid.New().String(),
		OwnerSlot:   string(cmd.IssuerSlot),
		Province:    prov.ID,
		UnitType:    kind.ID(),
		IssuedAt:    m.GameNow,
		CompletesAt: m.GameNow.Add(kind.BuildTime()),
	}
	m.NextRecruit[prov.ID] = order

	m.Timeline.Push(&timeline.Event{
		At:     order.CompletesAt,
		Kind:   timeline.RecruitComplete,
		UnitID: order.ID,
	})
	return "ok", nil, nil
}
