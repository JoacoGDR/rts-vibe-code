package matchdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/unitdom"
)

// resolveProvince handles the simplified combat model used for the MVP:
//  1. Collect every unit currently stationary in the province with HP > 0.
//  2. Stack units by coalition (Phase 4: alliance partners pool HP, no
//     friendly fire) so we end up with one champion per coalition.
//  3. While >1 coalition is present, deal damage from the strongest enemy
//     to the weakest champion using the rock-paper-scissors table.
//  4. The surviving coalition takes the province; the new owner is the
//     coalition member that contributed the strongest surviving stack.
func (m *Match) resolveProvince(prov *Province, at time.Time) []AppliedEvent {
	stationary := m.stationaryUnitsIn(prov.ID, at)
	if len(stationary) == 0 {
		return nil
	}
	champions := m.collapseChampions(stationary)

	out := m.fightUntilOneStands(champions, prov, at)
	out = append(out, m.captureIfOwnerChanged(prov, champions, at)...)
	return out
}

// stationaryUnitsIn collects every live unit that is currently stationary
// in the given province at time `at`.
func (m *Match) stationaryUnitsIn(provID string, at time.Time) []*Unit {
	out := []*Unit{}
	for _, u := range m.Units {
		if u.Origin == provID && u.Dest == provID && !u.IsMoving(at) && u.HP > 0 {
			out = append(out, u)
		}
	}
	return out
}

// collapseChampions returns one unit per coalition. Units belonging to
// allied slots are pooled (HP summed, weaker copies removed from the
// match). The champion's OwnerSlot is the coalition leader (lexically
// smallest slot id) so the rest of combat / capture logic can keep using
// `OwnerSlot` as the identity.
func (m *Match) collapseChampions(stationary []*Unit) map[string]*Unit {
	allSlots := m.SlotIDs()
	champions := map[string]*Unit{}
	for _, u := range stationary {
		coal := m.Diplomacy.CoalitionID(u.OwnerSlot, allSlots)
		if existing, ok := champions[coal]; ok {
			existing.HP += u.HP
			if u.HP > existing.HP/2 {
				existing.OwnerSlot = u.OwnerSlot
				existing.Type = u.Type
			}
			delete(m.Units, u.ID)
		} else {
			champions[coal] = u
		}
	}
	return champions
}

// fightUntilOneStands runs the combat loop, emitting damage / death
// events until at most one champion remains.
func (m *Match) fightUntilOneStands(champions map[string]*Unit, prov *Province, at time.Time) []AppliedEvent {
	out := []AppliedEvent{}
	for len(champions) > 1 {
		weakest := pickExtreme(champions, true)
		attacker := pickExtreme(excluding(champions, weakest), false)
		dmg := unitDamage(attacker.Type, weakest.Type)
		weakest.HP -= dmg
		m.Seq++
		out = append(out, AppliedEvent{
			Kind: "combat_damage", OccurAt: at, Seq: m.Seq,
			UnitID: weakest.ID, Slot: weakest.OwnerSlot,
			Extra: map[string]any{"hp": weakest.HP, "province": prov.ID},
		})
		if weakest.HP <= 0 {
			delete(m.Units, weakest.ID)
			delete(champions, weakest.OwnerSlot)
			m.Seq++
			out = append(out, AppliedEvent{
				Kind: "unit_destroyed", OccurAt: at, Seq: m.Seq,
				UnitID: weakest.ID, Slot: weakest.OwnerSlot,
				Extra: map[string]any{"province": prov.ID},
			})
		}
	}
	return out
}

// captureIfOwnerChanged transfers province ownership when a sole champion
// belongs to a different coalition than the previous owner. Re-flagging
// within the same coalition is a no-op (allies don't "steal" provinces
// from each other on garrison swaps).
func (m *Match) captureIfOwnerChanged(prov *Province, champions map[string]*Unit, at time.Time) []AppliedEvent {
	var winner *Unit
	for _, u := range champions {
		winner = u
	}
	if winner == nil {
		return nil
	}
	allSlots := m.SlotIDs()
	if prov.Owner != "" &&
		m.Diplomacy.CoalitionID(prov.Owner, allSlots) == m.Diplomacy.CoalitionID(winner.OwnerSlot, allSlots) {
		return nil
	}
	prev := prov.Owner
	prov.Owner = winner.OwnerSlot
	m.Seq++
	return []AppliedEvent{{
		Kind: "province_captured", OccurAt: at, Seq: m.Seq,
		Province: prov.ID, Slot: prov.Owner,
		Extra: map[string]any{"previous_owner": prev},
	}}
}

// pickExtreme returns the champion with the lowest (when low=true) or
// highest HP. The map must be non-empty.
func pickExtreme(champions map[string]*Unit, low bool) *Unit {
	var pick *Unit
	for _, u := range champions {
		if pick == nil || (low && u.HP < pick.HP) || (!low && u.HP > pick.HP) {
			pick = u
		}
	}
	return pick
}

// excluding returns a shallow copy of the map without the given unit.
func excluding(champions map[string]*Unit, skip *Unit) map[string]*Unit {
	out := make(map[string]*Unit, len(champions)-1)
	for k, u := range champions {
		if u == skip {
			continue
		}
		out[k] = u
	}
	return out
}

// unitDamage resolves the attacker→defender damage through the unitdom
// catalogue. Falls back to a flat 30 if either type is unknown.
func unitDamage(attacker, defender string) float64 {
	atk, ok1 := unitdom.ByID(attacker)
	def, ok2 := unitdom.ByID(defender)
	if !ok1 || !ok2 {
		return 30
	}
	return atk.DamageVs(def)
}
