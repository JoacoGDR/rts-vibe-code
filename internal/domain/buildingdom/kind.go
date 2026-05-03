package buildingdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

// Kind is the strategy interface a building type implements.
type Kind interface {
	ID() string
	Cost() economydom.Resources
	BuildTime() time.Duration
	Production() economydom.Resources // per macro pulse
}
