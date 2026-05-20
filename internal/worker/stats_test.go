package worker

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

func TestProjectStats(t *testing.T) {
	matchID := uuid.New()
	started := time.Now().Add(-10 * time.Minute)
	st := &wire.MatchState{
		MatchID:  matchID.String(),
		GameTime: time.Now(),
		Players: []wire.PlayerState{
			{ID: "red", Alive: true},
			{ID: "blue", Alive: false},
		},
		Units: []wire.UnitState{
			{ID: "u1", OwnerID: "red"},
			{ID: "u2", OwnerID: "red"},
		},
		Provinces: []wire.ProvinceState{
			{ID: "p1", OwnerID: "red", Capital: true},
		},
	}
	row := projectStats(matchID, &started, st)
	if row.TotalUnits != 2 {
		t.Fatalf("units: got %d want 2", row.TotalUnits)
	}
	if row.CapitalsTaken != 1 {
		t.Fatalf("capitals: got %d want 1", row.CapitalsTaken)
	}
	red, _ := row.PerSlot["red"].(map[string]any)
	if red == nil {
		t.Fatal("missing red per_slot")
	}
}
