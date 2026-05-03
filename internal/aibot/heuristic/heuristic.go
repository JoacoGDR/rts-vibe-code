package heuristic

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// MaxCommandsPerDecision is a hard cap so a buggy heuristic cannot
// drown the gateway/engine with commands per state envelope. The runner
// may further throttle in real time.
const MaxCommandsPerDecision = 6

// MinManpowerForRecruit is the bot's "I don't want to spend my last
// soldier on building a tank" threshold. The number is intentionally low
// — recruit cost decisions live in the engine.
const MinManpowerForRecruit = 12.0

// Decide is the heuristic's only entry point. It looks at the state the
// bot's slot perceives and returns the commands to enqueue this tick.
// `prng` makes choices reproducible in tests and lets us shuffle ties.
func Decide(state *wire.MatchState, ownSlot string, prng *rand.Rand) []wire.Command {
	if state == nil || ownSlot == "" {
		return nil
	}
	now := time.Now()
	cmds := make([]wire.Command, 0, MaxCommandsPerDecision)

	cmds = append(cmds, defendCapital(state, ownSlot, now, prng)...)
	cmds = append(cmds, recruitIfPossible(state, ownSlot, now, prng)...)
	cmds = append(cmds, dispatchIdleUnits(state, ownSlot, now, prng)...)
	cmds = append(cmds, diplomacyMoves(state, ownSlot, now)...)

	if len(cmds) > MaxCommandsPerDecision {
		cmds = cmds[:MaxCommandsPerDecision]
	}
	return cmds
}

// defendCapital makes sure the bot's capital always has at least one
// stationary unit on it. If we have no idle defender, recall the
// closest mover. Returning the implicit recruit happens via
// [recruitIfPossible].
func defendCapital(state *wire.MatchState, ownSlot string, now time.Time, prng *rand.Rand) []wire.Command {
	capital := findCapital(state, ownSlot)
	if capital == nil {
		return nil
	}
	garrisoned := false
	for i := range state.Units {
		u := &state.Units[i]
		if u.OwnerID != ownSlot {
			continue
		}
		if isOnProvince(u, capital) {
			garrisoned = true
			break
		}
	}
	if garrisoned {
		return nil
	}
	closest := closestOwnUnit(state, ownSlot, capital.X, capital.Y)
	if closest == nil || closest.Origin == capital.ID {
		return nil
	}
	return []wire.Command{newCommand(now, prng, "move", wire.Command{
		UnitID: closest.ID,
		From:   currentProvince(closest),
		To:     capital.ID,
	})}
}

// recruitIfPossible queues an infantry recruit at any owned province
// that has the resources to spare and no recruit already queued.
func recruitIfPossible(state *wire.MatchState, ownSlot string, now time.Time, prng *rand.Rand) []wire.Command {
	bank, ok := state.Resources[ownSlot]
	if !ok || bank.Manpower < MinManpowerForRecruit {
		return nil
	}
	queued := map[string]bool{}
	for _, q := range state.Queues.Recruits {
		if q.Owner == ownSlot {
			queued[q.Province] = true
		}
	}
	provs := ownedProvinces(state, ownSlot)
	if len(provs) == 0 {
		return nil
	}
	prng.Shuffle(len(provs), func(i, j int) { provs[i], provs[j] = provs[j], provs[i] })
	for _, p := range provs {
		if queued[p.ID] {
			continue
		}
		return []wire.Command{newCommand(now, prng, "recruit", wire.Command{
			From: p.ID,
			Args: map[string]string{"type": "infantry"},
		})}
	}
	return nil
}

// dispatchIdleUnits walks every idle owned unit and orders it toward the
// closest hostile-or-neutral neighbour province along the graph. The
// neighbour set is computed naively from the wire snapshot — units sit
// on their origin province while idle, so the heuristic uses province
// X/Y to find the closest enemy province visible to us.
func dispatchIdleUnits(state *wire.MatchState, ownSlot string, now time.Time, prng *rand.Rand) []wire.Command {
	cmds := []wire.Command{}
	hostile := hostileProvinces(state, ownSlot)
	for i := range state.Units {
		u := &state.Units[i]
		if u.OwnerID != ownSlot || isMoving(u, now) {
			continue
		}
		from := currentProvince(u)
		if from == "" {
			continue
		}
		target := closestHostileNeighbour(state, hostile, from)
		if target == "" {
			continue
		}
		cmds = append(cmds, newCommand(now, prng, "move", wire.Command{
			UnitID: u.ID, From: from, To: target,
		}))
		if len(cmds) >= MaxCommandsPerDecision {
			break
		}
	}
	return cmds
}

// diplomacyMoves emits at most one diplomacy command per decision: when
// the bot has fewer than half the surviving capitals it accepts any
// pending peace; otherwise it declares war on the leader if the bot is
// itself behind.
func diplomacyMoves(state *wire.MatchState, ownSlot string, now time.Time) []wire.Command {
	if len(state.Diplomacy) == 0 {
		return nil
	}
	for _, t := range state.Diplomacy {
		if t.Pending == "" || t.Pending == "alliance" {
			continue
		}
		if (t.SlotA == ownSlot || t.SlotB == ownSlot) && t.PendingFrom != ownSlot {
			return []wire.Command{{
				Kind:        "accept_peace",
				IssuedAt:    now,
				Idempotency: idem(now, "accept_peace", t.SlotA+t.SlotB),
				Args: map[string]string{
					"target_slot": otherSlot(ownSlot, t.SlotA, t.SlotB),
				},
			}}
		}
	}
	return nil
}

// findCapital returns the bot's capital province, or nil when missing.
func findCapital(state *wire.MatchState, ownSlot string) *wire.ProvinceState {
	for i := range state.Provinces {
		p := &state.Provinces[i]
		if p.OwnerID == ownSlot && p.Capital {
			return p
		}
	}
	return nil
}

// closestOwnUnit returns the bot's own unit closest to the given point,
// preferring stationary units when distances tie.
func closestOwnUnit(state *wire.MatchState, ownSlot string, x, y float64) *wire.UnitState {
	var best *wire.UnitState
	bestD := math.MaxFloat64
	for i := range state.Units {
		u := &state.Units[i]
		if u.OwnerID != ownSlot {
			continue
		}
		d := math.Hypot(u.X-x, u.Y-y)
		if d < bestD {
			best = u
			bestD = d
		}
	}
	return best
}

// ownedProvinces returns slice references to the bot's provinces.
func ownedProvinces(state *wire.MatchState, ownSlot string) []*wire.ProvinceState {
	out := []*wire.ProvinceState{}
	for i := range state.Provinces {
		p := &state.Provinces[i]
		if p.OwnerID == ownSlot {
			out = append(out, p)
		}
	}
	return out
}

// hostileProvinces flags every visible province whose owner is not the
// bot itself, not empty, and not at peace/alliance with the bot.
func hostileProvinces(state *wire.MatchState, ownSlot string) map[string]bool {
	stance := map[string]string{}
	for _, t := range state.Diplomacy {
		other := otherSlot(ownSlot, t.SlotA, t.SlotB)
		if other == "" {
			continue
		}
		stance[other] = t.Stance
	}
	out := map[string]bool{}
	for _, p := range state.Provinces {
		if p.OwnerID == "" || p.OwnerID == ownSlot {
			continue
		}
		if s := stance[p.OwnerID]; s == "peace" || s == "alliance" {
			continue
		}
		out[p.ID] = true
	}
	return out
}

// closestHostileNeighbour walks every other province on the snapshot and
// returns the closest hostile one (by euclidean distance), as a coarse
// proxy for the closest reachable hostile province. The engine still
// validates that an edge exists and rejects the command if not.
func closestHostileNeighbour(state *wire.MatchState, hostile map[string]bool, from string) string {
	if len(hostile) == 0 {
		return ""
	}
	src := provinceByID(state, from)
	if src == nil {
		return ""
	}
	type cand struct {
		id string
		d  float64
	}
	cands := []cand{}
	for i := range state.Provinces {
		p := &state.Provinces[i]
		if !hostile[p.ID] {
			continue
		}
		cands = append(cands, cand{p.ID, math.Hypot(p.X-src.X, p.Y-src.Y)})
	}
	if len(cands) == 0 {
		return ""
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].d < cands[j].d })
	return cands[0].id
}

func provinceByID(state *wire.MatchState, id string) *wire.ProvinceState {
	for i := range state.Provinces {
		if state.Provinces[i].ID == id {
			return &state.Provinces[i]
		}
	}
	return nil
}

func currentProvince(u *wire.UnitState) string {
	if u.Origin != "" {
		return u.Origin
	}
	return u.Dest
}

func isOnProvince(u *wire.UnitState, p *wire.ProvinceState) bool {
	if p == nil {
		return false
	}
	if u.Origin != "" && u.Origin == p.ID && (u.Dest == "" || u.Dest == p.ID) {
		return true
	}
	const eps = 0.5
	return math.Abs(u.X-p.X) < eps && math.Abs(u.Y-p.Y) < eps
}

func isMoving(u *wire.UnitState, now time.Time) bool {
	if u.ArrivesAt.IsZero() {
		return false
	}
	if u.Origin == u.Dest {
		return false
	}
	return now.Before(u.ArrivesAt)
}

func otherSlot(ownSlot, a, b string) string {
	switch ownSlot {
	case a:
		return b
	case b:
		return a
	}
	return ""
}

// newCommand stamps the issued-at and idempotency-key fields the
// gateway requires. Centralised so every command path uses the same
// scheme.
func newCommand(now time.Time, prng *rand.Rand, kind string, base wire.Command) wire.Command {
	base.Kind = kind
	base.IssuedAt = now
	base.Idempotency = idem(now, kind, randomTag(prng))
	return base
}

func randomTag(prng *rand.Rand) string {
	if prng == nil {
		return uuid.NewString()
	}
	var buf [12]byte
	for i := range buf {
		buf[i] = byte('a' + prng.IntN(26))
	}
	return string(buf[:])
}

func idem(now time.Time, kind, tag string) string {
	return kind + ":" + tag + ":" + now.UTC().Format("20060102T150405.000000Z")
}
