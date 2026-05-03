package visibility

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

func TestVisibilityHidesUnseenUnits(t *testing.T) {
	now := time.Now()
	m, err := matchdom.New("m1", "tiny-2p", map[string]string{"red": "u1", "blue": "u2"}, 60, now)
	if err != nil {
		t.Fatalf("new match: %v", err)
	}
	m.GameNow = now

	provincesRed, unitsRed := For(m, "red", now)
	if !provincesRed["A"] {
		t.Fatal("red should see its own capital A")
	}

	sawOwnUnit := false
	for id, u := range m.Units {
		if u.OwnerSlot == "red" && unitsRed[id] {
			sawOwnUnit = true
		}
	}
	if !sawOwnUnit {
		t.Fatal("red should see its own units")
	}

	for _, u := range m.Units {
		if u.OwnerSlot == "blue" {
			u.OriginX = 100000
			u.OriginY = 100000
			u.DestX = u.OriginX
			u.DestY = u.OriginY
		}
	}
	_, unitsRed = For(m, "red", now)
	for id, u := range m.Units {
		if u.OwnerSlot == "blue" && unitsRed[id] {
			t.Fatalf("red should not see far-away blue unit %s", id)
		}
	}
}
