package diplomacydom

import (
	"errors"
	"sort"
	"time"
)

// CooldownBetweenChanges is the minimum game-time between two diplomatic
// state changes between the same pair. Stops players from spamming
// declare-war / propose-peace.
const CooldownBetweenChanges = time.Hour

// Sentinel errors so handlers can errors.Is on stable values.
var (
	ErrSelfTreaty       = errors.New("cannot have a treaty with yourself")
	ErrCooldown         = errors.New("diplomatic action on cooldown")
	ErrNoPendingOffer   = errors.New("no matching pending offer to accept")
	ErrAlreadyAtStance  = errors.New("already at requested stance")
	ErrUnknownSlot      = errors.New("unknown slot")
	ErrPactAlreadyGiven = errors.New("pact already in place")
	ErrPactMissing      = errors.New("pact not in place")
)

// Registry is the in-memory diplomatic state for one match. Mutations
// must happen on the match-loop goroutine; the registry assumes single
// writer.
type Registry struct {
	treaties map[pairKey]*Treaty
	pacts    map[pactKey]*Pact
	version  uint64
}

// Version bumps on every mutation that can change vision contributors
// (treaties, alliances, pacts). [visibility] uses it to invalidate caches.
func (r *Registry) Version() uint64 {
	if r == nil {
		return 0
	}
	return r.version
}

func (r *Registry) bumpVersion() {
	if r != nil {
		r.version++
	}
}

// New returns an empty Registry. The engine creates one per match in
// matchdom.New and assumes the default stance for any unrepresented pair
// is [War] (see [Stance]).
func New() *Registry {
	return &Registry{
		treaties: map[pairKey]*Treaty{},
		pacts:    map[pactKey]*Pact{},
	}
}

type pairKey struct{ A, B string }
type pactKey struct {
	From, To string
	Kind     PactKind
}

func sortedPair(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}

// Stance reports the current diplomatic posture between a and b. The
// implicit default (when no treaty exists) is [War].
func (r *Registry) Stance(a, b string) Stance {
	if a == b {
		return Peace // a slot is always at peace with itself
	}
	x, y := sortedPair(a, b)
	t, ok := r.treaties[pairKey{x, y}]
	if !ok {
		return War
	}
	return t.Stance
}

// Treaty returns the raw treaty record for the pair, or nil if there is
// none. Callers usually want [Stance] instead.
func (r *Registry) Treaty(a, b string) *Treaty {
	if a == b {
		return nil
	}
	x, y := sortedPair(a, b)
	return r.treaties[pairKey{x, y}]
}

// AllTreaties returns every recorded treaty, sorted by (SlotA, SlotB) for
// deterministic output (snapshots, tests).
func (r *Registry) AllTreaties() []Treaty {
	out := make([]Treaty, 0, len(r.treaties))
	for _, t := range r.treaties {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SlotA != out[j].SlotA {
			return out[i].SlotA < out[j].SlotA
		}
		return out[i].SlotB < out[j].SlotB
	})
	return out
}

// PactsFrom returns every pact granted by `from`, regardless of recipient
// or kind. Used by visibility (share-map) and movement (right-of-way).
func (r *Registry) PactsFrom(from string) []Pact {
	out := []Pact{}
	for _, p := range r.pacts {
		if p.From == from {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// HasPact reports whether `from` has granted the given pact kind to `to`.
func (r *Registry) HasPact(from, to string, kind PactKind) bool {
	if from == to {
		return true
	}
	_, ok := r.pacts[pactKey{From: from, To: to, Kind: kind}]
	return ok
}

// IsHostile is a convenience used by combat: two slots fight only when
// their stance is War.
func (r *Registry) IsHostile(a, b string) bool { return r.Stance(a, b) == War }

// SeesThrough reports whether `viewer` is allowed to see what `owner`
// sees. True for Alliance and for any slot that has explicitly granted
// `viewer` a ShareMap pact.
func (r *Registry) SeesThrough(viewer, owner string) bool {
	if viewer == owner {
		return true
	}
	if r.Stance(viewer, owner) == Alliance {
		return true
	}
	return r.HasPact(owner, viewer, ShareMap)
}

// MayMoveThrough reports whether `mover` is allowed to enter (without
// fighting) a province owned by `owner`. True for empty/own provinces,
// Alliance partners, or when `owner` has granted RightOfWay to `mover`.
func (r *Registry) MayMoveThrough(mover, owner string) bool {
	if owner == "" || mover == owner {
		return true
	}
	switch r.Stance(mover, owner) {
	case Alliance:
		return true
	case Peace:
		return r.HasPact(owner, mover, RightOfWay)
	default:
		return false
	}
}

// Coalition returns every slot that shares an alliance with `slot`,
// transitively. The set always contains `slot` itself. This is the unit
// of "no friendly fire" and "shared victory".
func (r *Registry) Coalition(slot string, allSlots []string) map[string]bool {
	visited := map[string]bool{slot: true}
	stack := []string{slot}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, other := range allSlots {
			if visited[other] {
				continue
			}
			if r.Stance(current, other) == Alliance {
				visited[other] = true
				stack = append(stack, other)
			}
		}
	}
	return visited
}

// CoalitionID is a deterministic identifier for the coalition `slot`
// belongs to: the alphabetically-smallest slot in the coalition. Useful
// for chat scope keys and victory logging.
func (r *Registry) CoalitionID(slot string, allSlots []string) string {
	members := r.Coalition(slot, allSlots)
	leader := slot
	for s := range members {
		if s < leader {
			leader = s
		}
	}
	return leader
}
