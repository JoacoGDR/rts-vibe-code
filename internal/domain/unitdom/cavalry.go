package unitdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

type cavalry struct{}

func (cavalry) ID() string { return "cavalry" }
func (cavalry) Cost() economydom.Resources {
	return economydom.Resources{Manpower: 8, Food: 5, Iron: 3}
}
func (cavalry) BuildTime() time.Duration      { return 10 * time.Minute }
func (cavalry) Speed() float64                { return 100.0 / 3600.0 }
func (cavalry) StartingHP() float64           { return 80 }
func (cavalry) NeedsBuilding() string         { return "" }
func (k cavalry) DamageVs(other Kind) float64 { return damage(k, other) }
