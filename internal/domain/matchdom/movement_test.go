package matchdom_test

import (
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/pathdom"
)

func TestSetUnitPathAndPosition(t *testing.T) {
	m, err := matchdom.New("m1", "tiny-2p", map[string]string{"red": "u1", "blue": "u2"}, 1, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	var u *matchdom.Unit
	for _, unit := range m.Units {
		if unit.OwnerSlot == "red" {
			u = unit
			break
		}
	}
	if u == nil {
		t.Fatal("no red unit")
	}
	now := m.GameNow
	g := pathdom.NewGraph(m.Map)
	legs, err := g.Route(u.OriginX, u.OriginY, pathdom.Target{
		Kind: pathdom.TargetNode, Province: "B", X: 700, Y: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	u.Version++
	m.SetUnitPath(u, legs, now)
	if len(u.Path) == 0 {
		t.Fatal("expected path legs")
	}
	mid := now.Add(u.ArrivesAt.Sub(now) / 2)
	x, y := u.PositionAt(mid)
	if x == u.OriginX && y == u.OriginY {
		t.Fatalf("expected movement mid-leg, still at origin")
	}
}
