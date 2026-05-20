package pathdom

import (
	"math"

	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

const (
	NodeSnapRadius  = 24
	EdgeSnapMaxDist = 20
)

// Graph is the movement topology for a map (province centers + edges).
type Graph struct {
	m      *maps.Map
	coords map[string]vec2
	edges  []edgeSeg
}

type vec2 struct{ x, y float64 }

type edgeSeg struct {
	from, to       string
	ax, ay, bx, by float64
}

// NewGraph builds a pathfinding graph from a static map definition.
func NewGraph(m *maps.Map) *Graph {
	g := &Graph{
		m:      m,
		coords: make(map[string]vec2, len(m.Provinces)),
	}
	for _, p := range m.Provinces {
		g.coords[p.ID] = vec2{p.X, p.Y}
	}
	for _, e := range m.Edges {
		a, okA := g.coords[e.From]
		b, okB := g.coords[e.To]
		if !okA || !okB {
			continue
		}
		g.edges = append(g.edges, edgeSeg{
			from: e.From, to: e.To,
			ax: a.x, ay: a.y, bx: b.x, by: b.y,
		})
	}
	return g
}

func (g *Graph) coord(id string) (vec2, bool) {
	v, ok := g.coords[id]
	return v, ok
}

func (g *Graph) dist(a, b string) float64 {
	va, okA := g.coords[a]
	vb, okB := g.coords[b]
	if !okA || !okB {
		return math.MaxFloat64
	}
	return math.Hypot(va.x-vb.x, va.y-vb.y)
}
