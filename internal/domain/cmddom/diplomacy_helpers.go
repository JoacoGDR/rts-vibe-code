package cmddom

import (
	"errors"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

// resolveTarget reads the target slot from cmd.Args["target_slot"] and
// validates it exists in the match. Centralised so every diplomacy
// handler returns identical errors.
func resolveTarget(m *matchdom.Match, cmd Command) (string, string, error) {
	target, ok := cmd.Extra("target_slot")
	if !ok || target == "" {
		return "", "missing_target", ErrUnknownTarget
	}
	if _, ok := m.Players[target]; !ok {
		return "", "unknown_target", ErrUnknownTarget
	}
	if target == string(cmd.IssuerSlot) {
		return "", "self_target", ErrUnknownTarget
	}
	return target, "", nil
}

// translateDiplomacyErr converts a diplomacydom error into the (result,
// err) tuple every handler returns. Falls back to (action, ErrDiplomacy)
// for unrecognised errors so the metric still gets a sensible label.
func translateDiplomacyErr(err error, action string) (string, error) {
	switch {
	case errors.Is(err, diplomacydom.ErrSelfTreaty):
		return "self_target", ErrUnknownTarget
	case errors.Is(err, diplomacydom.ErrCooldown):
		return "cooldown", ErrDiplomacy
	case errors.Is(err, diplomacydom.ErrAlreadyAtStance):
		return "already_at_stance", ErrDiplomacy
	case errors.Is(err, diplomacydom.ErrNoPendingOffer):
		return "no_offer", ErrDiplomacy
	case errors.Is(err, diplomacydom.ErrPactAlreadyGiven):
		return "pact_already_given", ErrDiplomacy
	case errors.Is(err, diplomacydom.ErrPactMissing):
		return "pact_missing", ErrDiplomacy
	default:
		return action + "_failed", ErrDiplomacy
	}
}

// emitDiplomacyEvent builds the canonical AppliedEvent for a diplomacy
// state change so wire consumers (gateway, web client) see a uniform
// shape.
func emitDiplomacyEvent(m *matchdom.Match, cmd Command, target, kind string, extra map[string]any) []matchdom.AppliedEvent {
	m.Seq++
	if extra == nil {
		extra = map[string]any{}
	}
	extra["from_slot"] = string(cmd.IssuerSlot)
	extra["target_slot"] = target
	return []matchdom.AppliedEvent{{
		Kind: kind, OccurAt: m.GameNow, Seq: m.Seq,
		Slot: string(cmd.IssuerSlot), Extra: extra,
	}}
}
