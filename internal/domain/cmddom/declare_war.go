package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

type declareWarHandler struct{}

func (declareWarHandler) Kind() string { return "declare_war" }

func (declareWarHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.DeclareWar(string(cmd.IssuerSlot), target, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "declare_war")
		return code, nil, mapped
	}
	return "ok", emitDiplomacyEvent(m, cmd, target, "war_declared", nil), nil
}
