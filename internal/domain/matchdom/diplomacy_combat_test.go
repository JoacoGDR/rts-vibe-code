package matchdom_test

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
)

func newTinyMatch(t *testing.T) *matchdom.Match {
	t.Helper()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m, err := matchdom.New("m-coal", "tiny-2p",
		map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	m.GameNow = now
	return m
}

func newTriadMatch(t *testing.T) *matchdom.Match {
	t.Helper()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m, err := matchdom.New("m-triad", "tiny-3p",
		map[string]string{"red": "u1", "blue": "u2", "green": "u3"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	m.GameNow = now
	return m
}

func TestAlliesDoNotFightOnSameProvince(t *testing.T) {
	m := newTinyMatch(t)
	if err := m.Diplomacy.ProposeAlliance("red", "blue", m.GameNow); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if err := m.Diplomacy.AcceptAlliance("blue", "red", m.GameNow); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// Co-locate red and blue on province B (red's neighbour).
	for _, u := range m.Units {
		u.Origin = "B"
		u.Dest = "B"
	}
	prov := m.Provinces["B"]
	prov.Owner = "" // contested but currently neutral

	// Schedule and process an arrival event so the engine resolves combat.
	m.Timeline.Push(&timeline.Event{
		At: m.GameNow, Kind: timeline.Arrival,
	})
	for _, u := range m.Units {
		u.Version = 1
	}
	out := m.HandleEvent(&timeline.Event{
		At: m.GameNow, Kind: timeline.Arrival,
		UnitID: pickAnyUnitID(m, "red"), Version: 1,
	})
	_ = out

	// With allies pooling HP, both units should still be alive.
	for _, u := range m.Units {
		if u.HP <= 0 {
			t.Fatalf("expected allied units to survive together, got %s @0", u.OwnerSlot)
		}
	}
}

func TestCoalitionVictory(t *testing.T) {
	m := newTriadMatch(t)
	// Red and blue ally; green stands alone.
	_ = m.Diplomacy.ProposeAlliance("red", "blue", m.GameNow)
	_ = m.Diplomacy.AcceptAlliance("blue", "red", m.GameNow)

	// Strip green's capital. With three capitals on the map, removing
	// green's leaves only red+blue → coalition victory.
	for _, p := range m.Provinces {
		if p.Capital && p.Owner == "green" {
			p.Owner = "red"
		}
	}

	events := m.HandleEvent(&timeline.Event{
		At: m.GameNow, Kind: timeline.Arrival,
		UnitID: pickAnyUnitID(m, "red"), Version: 1,
	})
	if m.Status != "ended" {
		t.Fatalf("expected match ended, got status=%s", m.Status)
	}
	if len(m.WinnerCoal) != 2 {
		t.Fatalf("expected two winning slots, got %d (%v)", len(m.WinnerCoal), m.WinnerCoal)
	}
	hasMatchEnded := false
	for _, e := range events {
		if e.Kind == "match_ended" {
			hasMatchEnded = true
		}
	}
	if !hasMatchEnded {
		t.Fatal("expected match_ended event")
	}
}

func TestAllianceVisibilityShare(t *testing.T) {
	m := newTinyMatch(t)
	_ = m.Diplomacy.GrantPact("blue", "red", diplomacydom.ShareMap, m.GameNow)
	if !m.Diplomacy.SeesThrough("red", "blue") {
		t.Fatal("share-map grant must let red see through blue")
	}
}

func pickAnyUnitID(m *matchdom.Match, slot string) string {
	for id, u := range m.Units {
		if u.OwnerSlot == slot {
			return id
		}
	}
	return ""
}
