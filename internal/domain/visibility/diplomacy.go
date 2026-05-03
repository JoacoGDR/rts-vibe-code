package visibility

import (
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// TreatyStates projects the diplomatic registry onto the wire shape.
// Treaties are public information and the same payload is sent to every
// slot.
func TreatyStates(m *matchdom.Match) []wire.TreatyState {
	if m.Diplomacy == nil {
		return nil
	}
	all := m.Diplomacy.AllTreaties()
	out := make([]wire.TreatyState, 0, len(all))
	for _, t := range all {
		out = append(out, wire.TreatyState{
			SlotA:       t.SlotA,
			SlotB:       t.SlotB,
			Stance:      string(t.Stance),
			Pending:     string(t.Pending),
			PendingFrom: t.PendingFrom,
			ChangedAt:   t.ChangedAt,
		})
	}
	return out
}

// PactsFor returns the unilateral grants visible to `viewer`. When viewer
// is empty (snapshot path) every pact is returned. Otherwise we only
// include pacts where the viewer is the granter or the grantee — others
// must not learn that two enemies have a private treaty.
func PactsFor(m *matchdom.Match, viewer string) []wire.PactState {
	if m.Diplomacy == nil {
		return nil
	}
	out := []wire.PactState{}
	for _, p := range allPacts(m) {
		if viewer != "" && viewer != p.From && viewer != p.To {
			continue
		}
		out = append(out, wire.PactState{
			From: p.From, To: p.To,
			Kind:      string(p.Kind),
			GrantedAt: p.GrantedAt,
		})
	}
	return out
}

// allPacts walks every slot's outgoing pacts so callers don't have to.
func allPacts(m *matchdom.Match) []diplomacydom.Pact {
	out := []diplomacydom.Pact{}
	for _, slot := range m.SlotIDs() {
		out = append(out, m.Diplomacy.PactsFrom(slot)...)
	}
	return out
}
