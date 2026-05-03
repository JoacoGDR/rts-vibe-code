package economydom

// Resources is the per-slot stockpile. Phase 3 ships with three resources
// to keep recipes meaningful while the UI stays uncluttered:
//   - Manpower: spent to recruit any unit.
//   - Food:     consumed by units; runs out → morale penalty (Phase 4).
//   - Iron:     gates heavy units and factories.
type Resources struct {
	Manpower float64 `json:"manpower"`
	Food     float64 `json:"food"`
	Iron     float64 `json:"iron"`
}

// Add accumulates the components of o into r in-place.
func (r *Resources) Add(o Resources) {
	r.Manpower += o.Manpower
	r.Food += o.Food
	r.Iron += o.Iron
}

// CanPay reports whether r holds enough of every component to satisfy c.
func (r Resources) CanPay(c Resources) bool {
	return r.Manpower >= c.Manpower && r.Food >= c.Food && r.Iron >= c.Iron
}

// Pay deducts c from r in-place. Callers must check [CanPay] first.
func (r *Resources) Pay(c Resources) {
	r.Manpower -= c.Manpower
	r.Food -= c.Food
	r.Iron -= c.Iron
}

// Halved returns a copy of r with every component divided by two. Used
// for refunds when a queued order is invalidated.
func (r Resources) Halved() Resources {
	return Resources{Manpower: r.Manpower / 2, Food: r.Food / 2, Iron: r.Iron / 2}
}
