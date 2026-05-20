package maps_test

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

func TestArgentina2PLoad(t *testing.T) {
	m, err := maps.Load("argentina-2p")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Provinces) != 24 {
		t.Fatalf("provinces: got %d want 24", len(m.Provinces))
	}
	if len(m.Slots) != 2 {
		t.Fatalf("slots: got %d want 2", len(m.Slots))
	}
	for _, pair := range [][2]string{{"AR-S", "AR-X"}, {"AR-E", "AR-B"}, {"AR-A", "AR-J"}, {"AR-C", "AR-B"}} {
		if !m.HasEdge(pair[0], pair[1]) {
			t.Fatalf("missing frontier edge %s — %s", pair[0], pair[1])
		}
	}
}

func TestArgentina2PHomeTerritory(t *testing.T) {
	m, err := maps.Load("argentina-2p")
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{"north": 0, "south": 0}
	for _, p := range m.Provinces {
		if p.HomeSlot != "" {
			count[p.HomeSlot]++
		}
	}
	if count["north"] != 12 || count["south"] != 12 {
		t.Fatalf("home split: north=%d south=%d", count["north"], count["south"])
	}

	match, err := matchdom.New("m-ar", "argentina-2p", map[string]string{
		"north": "u-north",
		"south": "u-south",
	}, 1, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	owned := map[string]int{}
	for _, p := range match.Provinces {
		if p.Owner != "" {
			owned[p.Owner]++
		}
	}
	if owned["north"] != 12 || owned["south"] != 12 {
		t.Fatalf("match ownership: north=%d south=%d", owned["north"], owned["south"])
	}
}
