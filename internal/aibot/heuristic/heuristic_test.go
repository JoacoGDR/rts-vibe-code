package heuristic_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/aibot/heuristic"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// fixedPRNG returns a deterministic PRNG so test outputs don't drift.
func fixedPRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

// makeState returns a tiny two-slot battlefield with a configurable
// distribution of provinces. Both capitals are the slot's only owned
// province by default.
func makeState() *wire.MatchState {
	return &wire.MatchState{
		Provinces: []wire.ProvinceState{
			{ID: "p1", X: 0, Y: 0, OwnerID: "red", Capital: true},
			{ID: "p2", X: 100, Y: 0, OwnerID: "blue", Capital: true},
			{ID: "p3", X: 50, Y: 0},
		},
		Players: []wire.PlayerState{
			{ID: "red", Alive: true},
			{ID: "blue", Alive: true},
		},
		Resources: map[string]wire.ResourcePool{
			"red":  {Manpower: 30, Food: 30, Iron: 30},
			"blue": {Manpower: 30, Food: 30, Iron: 30},
		},
	}
}

func TestDecideNoStateNoCommands(t *testing.T) {
	if got := heuristic.Decide(nil, "red", fixedPRNG()); len(got) != 0 {
		t.Fatalf("expected no commands for nil state, got %d", len(got))
	}
	if got := heuristic.Decide(makeState(), "", fixedPRNG()); len(got) != 0 {
		t.Fatalf("expected no commands for empty slot, got %d", len(got))
	}
}

func TestDecideRecruitsWhenAffordable(t *testing.T) {
	st := makeState()
	cmds := heuristic.Decide(st, "red", fixedPRNG())
	hasRecruit := false
	for _, c := range cmds {
		if c.Kind == "recruit" {
			hasRecruit = true
			if c.From != "p1" {
				t.Fatalf("recruit must target capital, got %s", c.From)
			}
			if c.Args["type"] != "infantry" {
				t.Fatalf("recruit type expected infantry, got %s", c.Args["type"])
			}
		}
	}
	if !hasRecruit {
		t.Fatal("expected a recruit command when capital and resources are healthy")
	}
}

func TestDecideSkipsRecruitWhenBroke(t *testing.T) {
	st := makeState()
	st.Resources["red"] = wire.ResourcePool{Manpower: 1}
	for _, c := range heuristic.Decide(st, "red", fixedPRNG()) {
		if c.Kind == "recruit" {
			t.Fatal("should not recruit when manpower is low")
		}
	}
}

func TestDecideAttacksClosestHostile(t *testing.T) {
	st := makeState()
	st.Provinces = append(st.Provinces, wire.ProvinceState{
		ID: "p4", X: 200, Y: 0, OwnerID: "blue",
	})
	now := time.Now()
	st.Units = []wire.UnitState{
		{
			ID: "u1", OwnerID: "red", Type: "infantry",
			X: 0, Y: 0, HP: 100,
			Origin: "p1", Dest: "p1",
			StartedAt: now.Add(-time.Hour), ArrivesAt: now.Add(-time.Hour),
		},
	}
	cmds := heuristic.Decide(st, "red", fixedPRNG())
	var move *wire.Command
	for i := range cmds {
		if cmds[i].Kind == "move" && cmds[i].UnitID == "u1" {
			move = &cmds[i]
			break
		}
	}
	if move == nil {
		t.Fatal("expected a move command for the idle infantry")
	}
	if move.To != "p2" {
		t.Fatalf("expected to attack p2 (closest), got %s", move.To)
	}
}

func TestDecideRespectsPeace(t *testing.T) {
	st := makeState()
	st.Diplomacy = []wire.TreatyState{
		{SlotA: "blue", SlotB: "red", Stance: "peace"},
	}
	now := time.Now()
	st.Units = []wire.UnitState{
		{
			ID: "u1", OwnerID: "red",
			X: 0, Y: 0, Origin: "p1", Dest: "p1",
			StartedAt: now.Add(-time.Hour), ArrivesAt: now.Add(-time.Hour),
		},
	}
	for _, c := range heuristic.Decide(st, "red", fixedPRNG()) {
		if c.Kind == "move" {
			t.Fatalf("should not move during peace, got %+v", c)
		}
	}
}

func TestDecideAcceptsPeaceOffered(t *testing.T) {
	st := makeState()
	st.Diplomacy = []wire.TreatyState{
		{SlotA: "blue", SlotB: "red", Stance: "war", Pending: "peace", PendingFrom: "blue"},
	}
	cmds := heuristic.Decide(st, "red", fixedPRNG())
	hasAccept := false
	for _, c := range cmds {
		if c.Kind == "accept_peace" && c.Args["target_slot"] == "blue" {
			hasAccept = true
		}
	}
	if !hasAccept {
		t.Fatal("expected accept_peace when blue offers")
	}
}

func TestDecideSkipsRecruitWhenBusy(t *testing.T) {
	st := makeState()
	st.Queues = wire.QueueState{
		Recruits: []wire.QueuedRecruit{
			{Province: "p1", Owner: "red", UnitType: "infantry", CompletesAt: time.Now().Add(time.Hour)},
		},
	}
	for _, c := range heuristic.Decide(st, "red", fixedPRNG()) {
		if c.Kind == "recruit" && c.From == "p1" {
			t.Fatal("should not double-queue recruits at p1")
		}
	}
}

func TestDecideIdempotencyKeysAreUnique(t *testing.T) {
	st := makeState()
	cmds := heuristic.Decide(st, "red", fixedPRNG())
	seen := map[string]bool{}
	for _, c := range cmds {
		if seen[c.Idempotency] {
			t.Fatalf("duplicate idempotency key %s", c.Idempotency)
		}
		seen[c.Idempotency] = true
	}
}
