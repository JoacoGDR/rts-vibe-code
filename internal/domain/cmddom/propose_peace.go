package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

type proposePeaceHandler struct{}

func (proposePeaceHandler) Kind() string { return "propose_peace" }

func (proposePeaceHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.ProposePeace(string(cmd.IssuerSlot), target, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "propose_peace")
		return code, nil, mapped
	}
	extra := map[string]any{"pending": "peace"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "peace_proposed", extra), nil
}

type proposeAllianceHandler struct{}

func (proposeAllianceHandler) Kind() string { return "propose_alliance" }

func (proposeAllianceHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.ProposeAlliance(string(cmd.IssuerSlot), target, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "propose_alliance")
		return code, nil, mapped
	}
	extra := map[string]any{"pending": "alliance"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "alliance_proposed", extra), nil
}
