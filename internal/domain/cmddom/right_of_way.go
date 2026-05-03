package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

type rightOfWayGrantHandler struct{}

func (rightOfWayGrantHandler) Kind() string { return "right_of_way_grant" }

func (rightOfWayGrantHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.GrantPact(string(cmd.IssuerSlot), target, diplomacydom.RightOfWay, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "right_of_way_grant")
		return code, nil, mapped
	}
	extra := map[string]any{"pact": "right_of_way"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "pact_granted", extra), nil
}

type rightOfWayRevokeHandler struct{}

func (rightOfWayRevokeHandler) Kind() string { return "right_of_way_revoke" }

func (rightOfWayRevokeHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.RevokePact(string(cmd.IssuerSlot), target, diplomacydom.RightOfWay); err != nil {
		code, mapped := translateDiplomacyErr(err, "right_of_way_revoke")
		return code, nil, mapped
	}
	extra := map[string]any{"pact": "right_of_way"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "pact_revoked", extra), nil
}
