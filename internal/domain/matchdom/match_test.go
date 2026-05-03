package matchdom

import (
	"testing"
	"time"
)

func TestNewMatchSeedsProvincesAndPlayers(t *testing.T) {
	now := time.Now()
	m, err := New("m1", "tiny-2p", map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	if len(m.Provinces) == 0 {
		t.Fatal("expected provinces to be seeded")
	}
	if _, ok := m.Players["red"]; !ok {
		t.Fatal("expected red player to be seeded")
	}
	if m.Resources["red"].Manpower == 0 {
		t.Fatal("expected red to start with manpower")
	}
}

func TestMacroPulseAccrual(t *testing.T) {
	now := time.Now()
	m, err := New("m1", "tiny-2p", map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	m.GameNow = now.Add(time.Hour + time.Second)
	startManpower := m.Resources["red"].Manpower
	out := m.ProcessNext(m.GameNow)
	sawPulse := false
	for _, e := range out {
		if e.Kind == "macro_pulse" {
			sawPulse = true
		}
	}
	if !sawPulse {
		t.Fatal("expected macro_pulse event")
	}
	if m.Resources["red"].Manpower <= startManpower {
		t.Fatalf("expected manpower to grow, before=%f after=%f", startManpower, m.Resources["red"].Manpower)
	}
}
