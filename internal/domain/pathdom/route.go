package pathdom

import (
	"container/heap"
	"errors"
	"math"
)

// Leg is one straight segment along the movement graph.
type Leg struct {
	FromProv, ToProv string
	FromX, FromY     float64
	ToX, ToY         float64
}

var ErrNoRoute = errors.New("no route to target")

// Route plans a path from (fromX,fromY) to target using A* on provinces.
func (g *Graph) Route(fromX, fromY float64, target Target) ([]Leg, error) {
	startProv := g.nearestProvinceID(fromX, fromY)
	goalProv := target.goalProvince()

	if startProv == "" || goalProv == "" {
		return nil, ErrNoRoute
	}

	var via []string
	if startProv != goalProv {
		path, ok := g.astar(startProv, goalProv)
		if !ok {
			return nil, ErrNoRoute
		}
		via = path
	} else {
		via = []string{startProv}
	}

	legs := make([]Leg, 0, len(via))
	curX, curY := fromX, fromY
	curProv := startProv

	for i := 1; i < len(via); i++ {
		next := via[i]
		c, _ := g.coord(next)
		legs = append(legs, Leg{
			FromProv: curProv, ToProv: next,
			FromX: curX, FromY: curY, ToX: c.x, ToY: c.y,
		})
		curX, curY = c.x, c.y
		curProv = next
	}

	// Final leg to exact target coordinates.
	tx, ty := target.X, target.Y
	if len(legs) == 0 || legs[len(legs)-1].ToX != tx || legs[len(legs)-1].ToY != ty {
		legs = append(legs, Leg{
			FromProv: curProv, ToProv: goalProv,
			FromX: curX, FromY: curY, ToX: tx, ToY: ty,
		})
	}
	if len(legs) == 0 {
		legs = append(legs, Leg{
			FromProv: startProv, ToProv: goalProv,
			FromX: fromX, FromY: fromY, ToX: tx, ToY: ty,
		})
	}
	return legs, nil
}

func (t Target) goalProvince() string {
	if t.Kind == TargetNode {
		return t.Province
	}
	if t.T <= 0.5 {
		return t.EdgeFrom
	}
	return t.EdgeTo
}

type node struct {
	id       string
	f, g      float64
	parent   string
	index    int
	closed   bool
}

type pq []*node

func (h pq) Len() int           { return len(h) }
func (h pq) Less(i, j int) bool { return h[i].f < h[j].f }
func (h pq) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *pq) Push(x any) {
	n := x.(*node)
	n.index = len(*h)
	*h = append(*h, n)
}
func (h *pq) Pop() any {
	old := *h
	n := len(old) - 1
	item := old[n]
	old[n] = nil
	item.index = -1
	*h = old[:n]
	return item
}

func (g *Graph) astar(start, goal string) ([]string, bool) {
	open := &pq{}
	heap.Init(open)

	nodes := map[string]*node{start: {id: start, g: 0, f: g.dist(start, goal), index: -1}}
	heap.Push(open, nodes[start])

	for open.Len() > 0 {
		cur := heap.Pop(open).(*node)
		if cur.closed {
			continue
		}
		cur.closed = true
		if cur.id == goal {
			return reconstruct(nodes, goal), true
		}
		for _, nb := range g.m.Neighbors(cur.id) {
			tentG := cur.g + g.dist(cur.id, nb)
			nn, seen := nodes[nb]
			if !seen {
				nn = &node{id: nb, index: -1}
				nodes[nb] = nn
			}
			if nn.closed {
				continue
			}
			if seen && tentG >= nn.g {
				continue
			}
			nn.parent = cur.id
			nn.g = tentG
			nn.f = tentG + g.dist(nb, goal)
			if nn.index < 0 {
				heap.Push(open, nn)
			} else {
				heap.Fix(open, nn.index)
			}
		}
	}
	return nil, false
}

func reconstruct(nodes map[string]*node, goal string) []string {
	out := []string{goal}
	for cur := goal; ; {
		n := nodes[cur]
		if n == nil || n.parent == "" {
			break
		}
		cur = n.parent
		out = append(out, cur)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// LegLength returns the euclidean length of a leg.
func LegLength(l Leg) float64 {
	return math.Hypot(l.ToX-l.FromX, l.ToY-l.FromY)
}
