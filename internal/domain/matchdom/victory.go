package matchdom

import (
	"sort"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

// checkVictory inspects capital ownership and ends the match when only
// one coalition still controls a capital. The winning coalition's leader
// (alphabetically smallest slot in it) is recorded as `WinnerSlot` for
// backwards compatibility; every winning slot is listed in `WinnerCoal`
// so the wire layer can render an alliance victory.
//
// Idempotent: a no-op once the match status is anything other than
// "active".
func (m *Match) checkVictory(at time.Time) []AppliedEvent {
	if m.Status != "active" {
		return nil
	}
	capitalSlots := map[string]bool{}
	for _, p := range m.Provinces {
		if !p.Capital || p.Owner == "" {
			continue
		}
		capitalSlots[p.Owner] = true
	}
	if len(capitalSlots) == 0 {
		return nil
	}
	allSlots := m.SlotIDs()
	coalitions := map[string]map[string]bool{}
	for slot := range capitalSlots {
		id := m.Diplomacy.CoalitionID(slot, allSlots)
		if _, ok := coalitions[id]; !ok {
			coalitions[id] = m.Diplomacy.Coalition(slot, allSlots)
		}
	}
	if len(coalitions) != 1 {
		return nil
	}

	var (
		coalID  string
		members map[string]bool
	)
	for k, v := range coalitions {
		coalID = k
		members = v
	}
	winners := winningSlotList(members, capitalSlots)
	m.Status = "ended"
	m.WinnerSlot = coalID
	m.WinnerCoal = winners
	m.Seq++
	m.Timeline.Push(&timeline.Event{
		At:   at.Add(CleanupDelay),
		Kind: timeline.MatchCleanup,
	})
	return []AppliedEvent{{
		Kind: "match_ended", OccurAt: at, Seq: m.Seq, Slot: coalID,
		Extra: map[string]any{
			"coalition": coalID,
			"winners":   winners,
		},
	}}
}

// winningSlotList sorts coalition members that still hold a capital so
// the wire payload is deterministic (handy for tests and replays).
func winningSlotList(members, capitalSlots map[string]bool) []string {
	out := []string{}
	for s := range members {
		if capitalSlots[s] {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
