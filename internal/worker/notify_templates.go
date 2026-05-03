package worker

import (
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// template is the worker-internal mapping from one engine event to one
// notification kind. Targets is invoked with the live event and the
// match roster, returning the user ids that should be notified.
type template struct {
	Kind    notifydom.Kind
	Payload map[string]any
	Targets func(*wire.Event, []playerLookupResult) []uuid.UUID
}

// buildTemplates fans an event into zero-or-more templates. Returning
// nil short-circuits the dispatch loop without an extra allocation.
func buildTemplates(ev *wire.Event) []template {
	switch ev.Kind {
	case "province_captured":
		return capturedTemplates(ev)
	case "combat_damage":
		return underAttackTemplates(ev)
	case "treaty_proposed":
		return treatyTemplates(ev)
	case "match_ended":
		return matchEndedTemplates(ev)
	case "bot_takeover":
		return botTakeoverTemplates(ev)
	default:
		return nil
	}
}

func capturedTemplates(ev *wire.Event) []template {
	prevOwner, _ := ev.Extra["prev_owner"].(string)
	newOwner, _ := ev.Extra["new_owner"].(string)
	payload := map[string]any{
		"province":   ev.Province,
		"prev_owner": prevOwner,
		"new_owner":  newOwner,
	}
	return []template{{
		Kind:    notifydom.KindProvinceCaptured,
		Payload: payload,
		Targets: func(_ *wire.Event, players []playerLookupResult) []uuid.UUID {
			if prevOwner == "" {
				return nil
			}
			return userIDsForSlots(players, prevOwner)
		},
	}}
}

func underAttackTemplates(ev *wire.Event) []template {
	defender, _ := ev.Extra["defender"].(string)
	attacker, _ := ev.Extra["attacker"].(string)
	payload := map[string]any{
		"province": ev.Province,
		"defender": defender,
		"attacker": attacker,
	}
	return []template{{
		Kind:    notifydom.KindUnderAttack,
		Payload: payload,
		Targets: func(_ *wire.Event, players []playerLookupResult) []uuid.UUID {
			if defender == "" {
				return nil
			}
			return userIDsForSlots(players, defender)
		},
	}}
}

func treatyTemplates(ev *wire.Event) []template {
	from, _ := ev.Extra["from"].(string)
	to, _ := ev.Extra["to"].(string)
	stance, _ := ev.Extra["stance"].(string)
	payload := map[string]any{"from": from, "to": to, "stance": stance}
	return []template{{
		Kind:    notifydom.KindTreatyProposed,
		Payload: payload,
		Targets: func(_ *wire.Event, players []playerLookupResult) []uuid.UUID {
			if to == "" {
				return nil
			}
			return userIDsForSlots(players, to)
		},
	}}
}

func matchEndedTemplates(ev *wire.Event) []template {
	winners := stringList(ev.Extra["winners"])
	payload := map[string]any{"winners": winners}
	return []template{{
		Kind:    notifydom.KindMatchEnded,
		Payload: payload,
		Targets: func(_ *wire.Event, players []playerLookupResult) []uuid.UUID {
			out := make([]uuid.UUID, 0, len(players))
			for _, p := range players {
				out = append(out, p.UserID)
			}
			return out
		},
	}}
}

func botTakeoverTemplates(ev *wire.Event) []template {
	slot, _ := ev.Extra["slot"].(string)
	prevUserID, _ := ev.Extra["prev_user_id"].(string)
	payload := map[string]any{"slot": slot, "prev_user_id": prevUserID}
	return []template{{
		Kind:    notifydom.KindBotTakeover,
		Payload: payload,
		Targets: func(_ *wire.Event, _ []playerLookupResult) []uuid.UUID {
			id, err := uuid.Parse(prevUserID)
			if err != nil {
				return nil
			}
			return []uuid.UUID{id}
		},
	}}
}

func userIDsForSlots(players []playerLookupResult, slots ...string) []uuid.UUID {
	want := map[string]bool{}
	for _, s := range slots {
		if s != "" {
			want[s] = true
		}
	}
	out := []uuid.UUID{}
	for _, p := range players {
		if want[p.Slot] && p.Alive && !p.ControlledByAI {
			out = append(out, p.UserID)
		}
	}
	return out
}

// stringList accepts a JSON-decoded `extra["winners"]` field that
// pgrepo serialises as []any of strings. Returns the strings in order.
func stringList(v any) []string {
	out := []string{}
	switch s := v.(type) {
	case []string:
		return append(out, s...)
	case []any:
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
	}
	return out
}
