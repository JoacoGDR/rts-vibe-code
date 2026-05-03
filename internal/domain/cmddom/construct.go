package cmddom

import (
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/buildingdom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

type constructHandler struct{}

func (constructHandler) Kind() string { return "construct" }

func (constructHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	prov, ok := m.Provinces[string(cmd.From)]
	if !ok {
		return "bad_province", nil, ErrInvalidProvince
	}
	if prov.Owner != string(cmd.IssuerSlot) {
		return "unauthorized", nil, ErrUnauthorized
	}
	buildingType := string(cmd.UnitID)
	if t, ok := cmd.Extra("type"); ok {
		buildingType = t
	}
	kind, ok := buildingdom.ByID(buildingType)
	if !ok {
		return "bad_building_type", nil, ErrUnknownCommand
	}
	if matchdom.HasBuilding(m, prov.ID, kind.ID()) {
		return "already_built", nil, ErrAlreadyQueued
	}
	if _, busy := m.NextBuild[prov.ID]; busy {
		return "busy", nil, ErrAlreadyQueued
	}
	bank := m.Resources[string(cmd.IssuerSlot)]
	cost := kind.Cost()
	if bank == nil || !bank.CanPay(cost) {
		return "no_funds", nil, ErrInsufficientFunds
	}
	bank.Pay(cost)

	order := &matchdom.BuildOrder{
		ID:           uuid.New().String(),
		OwnerSlot:    string(cmd.IssuerSlot),
		Province:     prov.ID,
		BuildingType: kind.ID(),
		IssuedAt:     m.GameNow,
		CompletesAt:  m.GameNow.Add(kind.BuildTime()),
	}
	m.NextBuild[prov.ID] = order
	m.Timeline.Push(&timeline.Event{
		At:     order.CompletesAt,
		Kind:   timeline.ConstructComplete,
		UnitID: order.ID,
	})
	return "ok", nil, nil
}
