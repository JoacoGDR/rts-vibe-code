package visibility

import "math"

// visionCircle is a sight disc used when testing province/unit positions.
type visionCircle struct {
	x, y, r2 float64
}

// spatialIndex buckets circles on a uniform grid so point-in-circle tests
// only consider nearby discs instead of every contributor circle.
type spatialIndex struct {
	cellSize float64
	minX     float64
	minY     float64
	cols     int
	rows     int
	cells    [][]int
}

func newSpatialIndex(circles []visionCircle, points [][2]float64) *spatialIndex {
	if len(circles) == 0 {
		return nil
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	maxR := 0.0
	for _, c := range circles {
		r := math.Sqrt(c.r2)
		if r > maxR {
			maxR = r
		}
		if c.x-r < minX {
			minX = c.x - r
		}
		if c.x+r > maxX {
			maxX = c.x + r
		}
		if c.y-r < minY {
			minY = c.y - r
		}
		if c.y+r > maxY {
			maxY = c.y + r
		}
	}
	for _, p := range points {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	if len(points) == 0 && minX == math.MaxFloat64 {
		return nil
	}
	cellSize := maxR
	if cellSize < 1 {
		cellSize = 1
	}
	spanX := maxX - minX
	spanY := maxY - minY
	if spanX < cellSize {
		spanX = cellSize
	}
	if spanY < cellSize {
		spanY = cellSize
	}
	cols := int(spanX/cellSize) + 1
	rows := int(spanY/cellSize) + 1
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	idx := &spatialIndex{
		cellSize: cellSize,
		minX:     minX,
		minY:     minY,
		cols:     cols,
		rows:     rows,
		cells:    make([][]int, cols*rows),
	}
	for i, c := range circles {
		r := math.Sqrt(c.r2)
		c0 := int((c.x - r - minX) / cellSize)
		c1 := int((c.x + r - minX) / cellSize)
		r0 := int((c.y - r - minY) / cellSize)
		r1 := int((c.y + r - minY) / cellSize)
		if c0 < 0 {
			c0 = 0
		}
		if r0 < 0 {
			r0 = 0
		}
		if c1 >= cols {
			c1 = cols - 1
		}
		if r1 >= rows {
			r1 = rows - 1
		}
		for col := c0; col <= c1; col++ {
			for row := r0; row <= r1; row++ {
				idx.cells[col+row*cols] = append(idx.cells[col+row*cols], i)
			}
		}
	}
	return idx
}

func (idx *spatialIndex) within(px, py float64, circles []visionCircle) bool {
	if idx == nil {
		return false
	}
	col := int((px - idx.minX) / idx.cellSize)
	row := int((py - idx.minY) / idx.cellSize)
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			c := col + dc
			r := row + dr
			if c < 0 || r < 0 || c >= idx.cols || r >= idx.rows {
				continue
			}
			for _, i := range idx.cells[c+r*idx.cols] {
				circ := circles[i]
				dx := px - circ.x
				dy := py - circ.y
				if dx*dx+dy*dy <= circ.r2 {
					return true
				}
			}
		}
	}
	return false
}
