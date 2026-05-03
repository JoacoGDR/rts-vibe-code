package unitdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

type armor struct{}

func (armor) ID() string { return "armor" }
func (armor) Cost() economydom.Resources {
	return economydom.Resources{Manpower: 12, Food: 5, Iron: 8}
}
func (armor) BuildTime() time.Duration      { return 20 * time.Minute }
func (armor) Speed() float64                { return 80.0 / 3600.0 }
func (armor) StartingHP() float64           { return 140 }
func (armor) NeedsBuilding() string         { return "factory" }
func (k armor) DamageVs(other Kind) float64 { return damage(k, other) }
