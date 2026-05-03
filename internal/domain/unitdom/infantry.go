package unitdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

type infantry struct{}

func (infantry) ID() string                    { return "infantry" }
func (infantry) Cost() economydom.Resources    { return economydom.Resources{Manpower: 5, Food: 3} }
func (infantry) BuildTime() time.Duration      { return 5 * time.Minute }
func (infantry) Speed() float64                { return 50.0 / 3600.0 }
func (infantry) StartingHP() float64           { return 100 }
func (infantry) NeedsBuilding() string         { return "" }
func (k infantry) DamageVs(other Kind) float64 { return damage(k, other) }
