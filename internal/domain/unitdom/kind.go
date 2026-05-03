package unitdom

import (
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
)

// Kind is the strategy interface a single unit type implements. Every
// behaviour the simulation needs to know about a unit (cost, build time,
// movement speed, combat numbers, prerequisites) is a method on Kind so
// no caller ever has to type-switch on a string.
type Kind interface {
	ID() string
	Cost() economydom.Resources
	BuildTime() time.Duration
	Speed() float64 // pixels / game-second
	StartingHP() float64
	NeedsBuilding() string // "" if no prerequisite
	DamageVs(other Kind) float64
}

// damageTable encodes the rock-paper-scissors balance shared by every Kind:
//
//	infantry > armor > cavalry > infantry
//
// It is keyed by Kind.ID() because the table is shared across types — Go
// does not let you build cross-receiver generic methods cleanly otherwise.
var damageTable = map[string]map[string]float64{
	"infantry": {"infantry": 30, "cavalry": 22, "armor": 38},
	"cavalry":  {"infantry": 38, "cavalry": 30, "armor": 22},
	"armor":    {"infantry": 22, "cavalry": 38, "armor": 30},
}

const defaultDamage = 30.0

func damage(attacker, defender Kind) float64 {
	if row, ok := damageTable[attacker.ID()]; ok {
		if v, ok := row[defender.ID()]; ok {
			return v
		}
	}
	return defaultDamage
}
