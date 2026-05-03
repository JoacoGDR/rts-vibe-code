package cmddom_test

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/cmddom"
	"github.com/joaquing/clone-supremacy/internal/domain/ids"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

func newMatch(t *testing.T, now time.Time) *matchdom.Match {
	t.Helper()
	m, err := matchdom.New("m1", "tiny-2p", map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	m.GameNow = now
	return m
}

func TestMoveCapturesEnemyCapital(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := newMatch(t, now)

	var redUnit string
	for id, u := range m.Units {
		if u.OwnerSlot == "red" {
			redUnit = id
		}
	}
	if redUnit == "" {
		t.Fatal("no red unit")
	}

	for id, u := range m.Units {
		if u.OwnerSlot == "blue" {
			delete(m.Units, id)
			_ = u
		}
	}

	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1",
		Kind: "move", UnitID: ids.UnitID(redUnit), From: "A", To: "D",
		IssuedAt: now,
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	u := m.Units[redUnit]
	if u.Dest != "D" {
		t.Fatalf("expected dest D, got %s", u.Dest)
	}

	m.GameNow = u.ArrivesAt
	events := m.ProcessNext(m.GameNow)
	if len(events) == 0 {
		t.Fatal("expected events on arrival")
	}
	if m.Status != "ended" {
		t.Fatalf("match should have ended, status=%s", m.Status)
	}
	if m.WinnerSlot != "red" {
		t.Fatalf("expected red winner, got %s", m.WinnerSlot)
	}
}

func TestUnauthorisedMoveRejected(t *testing.T) {
	now := time.Now()
	m := newMatch(t, now)
	var blueUnit string
	for id, u := range m.Units {
		if u.OwnerSlot == "blue" {
			blueUnit = id
			break
		}
	}
	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1", IssuerSlot: "red",
		Kind: "move", UnitID: ids.UnitID(blueUnit), From: "D", To: "B",
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestNonAdjacentMoveRejected(t *testing.T) {
	now := time.Now()
	m := newMatch(t, now)
	var redUnit string
	for id, u := range m.Units {
		if u.OwnerSlot == "red" {
			redUnit = id
			break
		}
	}
	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1", IssuerSlot: "red",
		Kind: "move", UnitID: ids.UnitID(redUnit), From: "A", To: "Z",
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err == nil {
		t.Fatal("expected invalid dest error")
	}
}

func TestStaleArrivalDropped(t *testing.T) {
	now := time.Now()
	m := newMatch(t, now)

	var unitID string
	for id, u := range m.Units {
		if u.OwnerSlot == "red" {
			unitID = id
			break
		}
	}
	if unitID == "" {
		t.Fatal("no red unit")
	}

	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1", IssuerSlot: "red",
		Kind: "move", UnitID: ids.UnitID(unitID), From: "A", To: "B",
		IssuedAt: now,
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	var arrival *timeline.Event
	for {
		ev := m.Timeline.Pop()
		if ev == nil {
			break
		}
		if ev.Kind == timeline.Arrival {
			arrival = ev
			break
		}
	}
	if arrival == nil {
		t.Fatal("never saw arrival event")
	}
	arrival.Version = 999
	out := m.HandleEvent(arrival)
	if len(out) != 0 {
		t.Fatalf("stale event should produce no events, got %v", out)
	}
}

func TestRecruitSpendsResourcesAndQueuesUnit(t *testing.T) {
	now := time.Now()
	m := newMatch(t, now)
	startCount := len(m.Units)

	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1", IssuerSlot: "red",
		Kind: "recruit", From: "A",
		Args: map[string]string{"type": "infantry"},
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err != nil {
		t.Fatalf("recruit: %v", err)
	}
	if _, ok := m.NextRecruit["A"]; !ok {
		t.Fatal("expected recruit to be queued for A")
	}
	if m.Resources["red"].Manpower != 25 {
		t.Fatalf("expected 25 manpower, got %f", m.Resources["red"].Manpower)
	}
	if len(m.Units) != startCount {
		t.Fatal("recruit must not spawn unit synchronously")
	}

	order := m.NextRecruit["A"]
	m.GameNow = order.CompletesAt
	events := m.ProcessNext(m.GameNow)
	hasRecruited := false
	for _, e := range events {
		if e.Kind == "unit_recruited" {
			hasRecruited = true
		}
	}
	if !hasRecruited {
		t.Fatal("expected unit_recruited event")
	}
	if len(m.Units) != startCount+1 {
		t.Fatalf("expected one new unit, got %d total", len(m.Units))
	}
}

func TestArmorRequiresFactory(t *testing.T) {
	now := time.Now()
	m := newMatch(t, now)
	cmd := cmddom.Command{
		MatchID: "m1", UserID: "u1", IssuerSlot: "red",
		Kind: "recruit", From: "A",
		Args: map[string]string{"type": "armor"},
	}
	if _, _, err := cmddom.Dispatch(m, cmd); err == nil {
		t.Fatal("armor recruit without factory should fail")
	}
}
