package pathdom_test

import (
	"testing"

	"github.com/joaquing/clone-supremacy/internal/domain/pathdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

func tiny2p(t *testing.T) *pathdom.Graph {
	t.Helper()
	m, err := maps.Load("tiny-2p")
	if err != nil {
		t.Fatal(err)
	}
	return pathdom.NewGraph(m)
}

func TestSnapNode(t *testing.T) {
	g := tiny2p(t)
	target := g.SnapTarget(100, 100)
	if target.Kind != pathdom.TargetNode || target.Province != "A" {
		t.Fatalf("snap node: %+v", target)
	}
}

func TestSnapEdge(t *testing.T) {
	g := tiny2p(t)
	// Midpoint of A-B horizontal edge (centers 100,100 and 700,100).
	target := g.SnapTarget(400, 100)
	if target.Kind != pathdom.TargetEdge {
		t.Fatalf("expected edge snap, got %+v", target)
	}
	if target.EdgeFrom != "A" || target.EdgeTo != "B" {
		t.Fatalf("edge ids: %+v", target)
	}
}

func TestRouteMultiHop(t *testing.T) {
	g := tiny2p(t)
	// A to D via graph (not direct if only one hop - A-D exists in yaml).
	target := pathdom.Target{Kind: pathdom.TargetNode, Province: "D", X: 700, Y: 500}
	legs, err := g.Route(100, 100, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(legs) < 1 {
		t.Fatal("expected at least one leg")
	}
	last := legs[len(legs)-1]
	if last.ToX != 700 || last.ToY != 500 {
		t.Fatalf("last leg end: %+v", last)
	}
}

func TestRouteEdgeTarget(t *testing.T) {
	g := tiny2p(t)
	target := g.SnapTarget(400, 100)
	legs, err := g.Route(100, 100, target)
	if err != nil {
		t.Fatal(err)
	}
	last := legs[len(legs)-1]
	if last.ToX != target.X || last.ToY != target.Y {
		t.Fatalf("final coords %v,%v want %v,%v", last.ToX, last.ToY, target.X, target.Y)
	}
}

// TestRouteFromEdgeMidFlight verifies that a unit mid-flight on edge A-B routes
// correctly to its destination even when its current position is geometrically
// closer to a third province C that is not on the traversed edge.
//
// Graph topology:
//
//	A(0,0) --[edge]--> B(200,0) --[edge]--> D(200,200)
//	C(80,5) --[edge]--> A  (C is near the A-B midpoint but not on A-B)
//
// Unit position: (120, 0) — on edge A-B, closer to B than A.
// Without edge hints, nearestProvinceID resolves to C (dist approx 20), and A*
// returns a route that starts from C — routing the unit through an incorrect
// province it never entered.
// RouteFromEdge with edgeFrom=A, edgeTo=B correctly starts from B and
// routes B-D directly without detouring through C or A.
func TestRouteFromEdgeMidFlight(t *testing.T) {
	m := &maps.Map{
		ID: "test",
		Provinces: []maps.Province{
			{ID: "A", X: 0, Y: 0},
			{ID: "B", X: 200, Y: 0},
			{ID: "C", X: 80, Y: 5},
			{ID: "D", X: 200, Y: 200},
		},
		Edges: []maps.Edge{
			{From: "A", To: "B"},
			{From: "B", To: "D"},
			{From: "C", To: "A"},
		},
	}
	g := pathdom.NewGraph(m)

	unitX, unitY := 120.0, 0.0
	destination := pathdom.Target{Kind: pathdom.TargetNode, Province: "D", X: 200, Y: 200}

	// Route without edge hints starts from C (nearest province center, dist approx 20)
	// and routes C-A-B-D, incorrectly treating the unit as if it is at C.
	wrongLegs, err := g.Route(unitX, unitY, destination)
	if err != nil {
		t.Fatalf("Route (without hints) unexpectedly failed: %v", err)
	}
	if len(wrongLegs) == 0 || wrongLegs[0].FromProv != "C" {
		t.Fatalf("expected Route to (incorrectly) start from C, got first leg from %q", wrongLegs[0].FromProv)
	}

	// RouteFromEdge with the unit's actual edge (A-B) starts from B
	// (the closer endpoint at dist=80 vs A at dist=120) and finds B-D directly.
	legs, err := g.RouteFromEdge(unitX, unitY, "A", "B", destination)
	if err != nil {
		t.Fatalf("RouteFromEdge failed: %v", err)
	}
	if len(legs) == 0 {
		t.Fatal("expected legs, got none")
	}
	if legs[0].FromProv != "B" {
		t.Fatalf("expected route to start from B (closer edge endpoint), got %q", legs[0].FromProv)
	}
	last := legs[len(legs)-1]
	if last.ToProv != "D" {
		t.Fatalf("expected last leg to arrive at D, got %q", last.ToProv)
	}
}
