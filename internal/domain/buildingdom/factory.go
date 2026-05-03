package buildingdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

type factory struct{}

func (factory) ID() string { return "factory" }
func (factory) Cost() economydom.Resources {
	return economydom.Resources{Iron: 20, Manpower: 10}
}
func (factory) BuildTime() time.Duration { return 30 * time.Minute }
func (factory) Production() economydom.Resources {
	return economydom.Resources{Iron: 4, Manpower: 2}
}
