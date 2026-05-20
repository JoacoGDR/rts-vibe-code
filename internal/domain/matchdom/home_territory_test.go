package matchdom_test

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

func TestHomeTerritoryOwnership(t *testing.T) {
	m, err := maps.Load("classic-4p")
	if err != nil {
		t.Fatal(err)
	}
	countBySlot := map[string]int{}
	for _, p := range m.Provinces {
		if p.HomeSlot != "" {
			countBySlot[p.HomeSlot]++
		}
	}
	for _, slot := range []string{"red", "blue", "green", "yellow"} {
		if countBySlot[slot] != 8 {
			t.Fatalf("slot %s: want 8 home provinces, got %d", slot, countBySlot[slot])
		}
	}

	assign := map[string]string{
		"red": uuidSlot("u-red"), "blue": uuidSlot("u-blue"),
		"green": uuidSlot("u-green"), "yellow": uuidSlot("u-yellow"),
	}
	match, err := matchdom.New("m1", "classic-4p", assign, 1, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	owned := map[string]int{}
	for _, p := range match.Provinces {
		if p.Owner != "" {
			owned[p.Owner]++
		}
	}
	for _, slot := range []string{"red", "blue", "green", "yellow"} {
		if owned[slot] != 8 {
			t.Fatalf("match slot %s: want 8 owned provinces, got %d", slot, owned[slot])
		}
	}
}

func uuidSlot(s string) string { return s }
