package balance

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

// MacroPulseInterval is the game-time gap between economic ticks. The
// engine schedules one [timeline.MacroPulse] per slot every interval.
const MacroPulseInterval = time.Hour

// StartingResources returns the resource bank a slot owns at match start.
// Tuned to allow recruiting one infantry plus a small buffer.
func StartingResources() economydom.Resources {
	return economydom.Resources{Manpower: 30, Food: 50, Iron: 20}
}

// ProvinceRevenue is the per-pulse trickle a slot earns for each province
// they own.
func ProvinceRevenue() economydom.Resources {
	return economydom.Resources{Manpower: 4, Food: 5, Iron: 1}
}

// View radii in pixel-units, used by the visibility computation.
const (
	OwnedProvinceViewRadius = 180.0
	DefaultUnitViewRadius   = 200.0
)

// UnitViewRadius returns the per-unit-type sight range. Falls back to
// [DefaultUnitViewRadius] for anything not in the table.
func UnitViewRadius(unitType string) float64 {
	switch unitType {
	case "infantry":
		return 220
	case "cavalry":
		return 280
	case "armor":
		return 260
	default:
		return DefaultUnitViewRadius
	}
}

// FallbackUnitSpeed is the speed assigned to a unit whose type is not in
// the [unitdom] registry. Keeps the simulation deterministic even when the
// catalogue is mid-edit.
const FallbackUnitSpeed = 50.0 / 3600.0
