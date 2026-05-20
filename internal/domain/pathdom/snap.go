package pathdom

import "math"

// TargetKind distinguishes province-center vs edge-param targets.
type TargetKind string

const (
	TargetNode TargetKind = "node"
	TargetEdge TargetKind = "edge"
)

// Target is a movement destination on the graph.
type Target struct {
	Kind     TargetKind
	Province string
	EdgeFrom string
	EdgeTo   string
	X, Y     float64
	T        float64 // param along edge segment [0,1] from EdgeFrom center to EdgeTo center
}

// SnapTarget picks the nearest valid node or edge point to (x,y).
func (g *Graph) SnapTarget(x, y float64) Target {
	bestNode := ""
	bestNodeDist := NodeSnapRadius + 1.0
	for id, c := range g.coords {
		d := math.Hypot(x-c.x, y-c.y)
		if d <= NodeSnapRadius && d < bestNodeDist {
			bestNodeDist = d
			bestNode = id
		}
	}
	if bestNode != "" {
		c := g.coords[bestNode]
		return Target{Kind: TargetNode, Province: bestNode, X: c.x, Y: c.y}
	}

	var best edgeSeg
	bestDist := float64(EdgeSnapMaxDist) + 1
	bestT := 0.0
	bestX, bestY := x, y
	for _, e := range g.edges {
		px, py, t, d := projectSegment(x, y, e.ax, e.ay, e.bx, e.by)
		if d <= EdgeSnapMaxDist && d < bestDist {
			best = e
			bestDist = d
			bestT = t
			bestX, bestY = px, py
		}
	}
	if bestDist <= EdgeSnapMaxDist {
		return Target{
			Kind: TargetEdge, EdgeFrom: best.from, EdgeTo: best.to,
			X: bestX, Y: bestY, T: bestT,
		}
	}

	// Fallback: nearest province center.
	near := g.nearestProvinceID(x, y)
	c := g.coords[near]
	return Target{Kind: TargetNode, Province: near, X: c.x, Y: c.y}
}

func (g *Graph) nearestProvinceID(x, y float64) string {
	best := ""
	bestD := math.MaxFloat64
	for id, c := range g.coords {
		d := math.Hypot(x-c.x, y-c.y)
		if d < bestD {
			bestD = d
			best = id
		}
	}
	return best
}

func projectSegment(px, py, ax, ay, bx, by float64) (x, y, t, dist float64) {
	dx := bx - ax
	dy := by - ay
	len2 := dx*dx + dy*dy
	if len2 == 0 {
		return ax, ay, 0, math.Hypot(px-ax, py-ay)
	}
	t = ((px-ax)*dx + (py-ay)*dy) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	x = ax + t*dx
	y = ay + t*dy
	return x, y, t, math.Hypot(px-x, py-y)
}
