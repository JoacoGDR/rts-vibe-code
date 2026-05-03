package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

type acceptPeaceHandler struct{}

func (acceptPeaceHandler) Kind() string { return "accept_peace" }

func (acceptPeaceHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.AcceptPeace(string(cmd.IssuerSlot), target, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "accept_peace")
		return code, nil, mapped
	}
	return "ok", emitDiplomacyEvent(m, cmd, target, "peace_signed", nil), nil
}

type acceptAllianceHandler struct{}

func (acceptAllianceHandler) Kind() string { return "accept_alliance" }

func (acceptAllianceHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.AcceptAlliance(string(cmd.IssuerSlot), target, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "accept_alliance")
		return code, nil, mapped
	}
	extra := map[string]any{
		"coalition": m.Diplomacy.CoalitionID(string(cmd.IssuerSlot), m.SlotIDs()),
	}
	return "ok", emitDiplomacyEvent(m, cmd, target, "alliance_signed", extra), nil
}
