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
