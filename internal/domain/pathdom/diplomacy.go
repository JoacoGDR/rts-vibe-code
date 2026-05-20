package pathdom

import (
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
)

// ProvinceOwners maps province id -> owner slot (empty = neutral).
type ProvinceOwners map[string]string

// LegBlocked reports whether mover may enter the leg's destination province.
func LegBlocked(d *diplomacydom.Registry, mover string, owners ProvinceOwners, leg Leg) bool {
	owner := owners[leg.ToProv]
	if owner == "" || owner == mover {
		return false
	}
	if d.MayMoveThrough(mover, owner) {
		return false
	}
	return d.Stance(mover, owner) != diplomacydom.War
}

// HostileLegEnd is true when the leg ends in enemy territory at war.
func HostileLegEnd(d *diplomacydom.Registry, mover string, owners ProvinceOwners, leg Leg) bool {
	owner := owners[leg.ToProv]
	if owner == "" || owner == mover {
		return false
	}
	return d.Stance(mover, owner) == diplomacydom.War
}

// RouteAttackTerminus is true if any leg ends in hostile territory.
func RouteAttackTerminus(d *diplomacydom.Registry, mover string, owners ProvinceOwners, legs []Leg) bool {
	for _, l := range legs {
		if HostileLegEnd(d, mover, owners, l) {
			return true
		}
	}
	return false
}

// RouteBlocked returns true if any leg is blocked by diplomacy (non-war).
func RouteBlocked(d *diplomacydom.Registry, mover string, owners ProvinceOwners, legs []Leg) bool {
	for _, l := range legs {
		if LegBlocked(d, mover, owners, l) {
			return true
		}
	}
	return false
}
