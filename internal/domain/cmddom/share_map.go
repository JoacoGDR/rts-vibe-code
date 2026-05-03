package cmddom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

type shareMapHandler struct{}

func (shareMapHandler) Kind() string { return "share_map" }

func (shareMapHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.GrantPact(string(cmd.IssuerSlot), target, diplomacydom.ShareMap, m.GameNow); err != nil {
		code, mapped := translateDiplomacyErr(err, "share_map")
		return code, nil, mapped
	}
	extra := map[string]any{"pact": "share_map"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "pact_granted", extra), nil
}

type revokeShareMapHandler struct{}

func (revokeShareMapHandler) Kind() string { return "revoke_share_map" }

func (revokeShareMapHandler) Apply(m *matchdom.Match, cmd Command) (string, []matchdom.AppliedEvent, error) {
	target, code, err := resolveTarget(m, cmd)
	if err != nil {
		return code, nil, err
	}
	if err := m.Diplomacy.RevokePact(string(cmd.IssuerSlot), target, diplomacydom.ShareMap); err != nil {
		code, mapped := translateDiplomacyErr(err, "revoke_share_map")
		return code, nil, mapped
	}
	extra := map[string]any{"pact": "share_map"}
	return "ok", emitDiplomacyEvent(m, cmd, target, "pact_revoked", extra), nil
}
