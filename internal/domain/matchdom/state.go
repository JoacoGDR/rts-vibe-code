package matchdom

import (
	"math"
	"sort"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
	"github.com/joaquing/clone-supremacy/internal/domain/pathdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// Unit is the authoritative simulation entity. All positions and
// timestamps are in server "game time" — the simulated wall-clock time,
// advanced at SpeedFactor × real time. Movement is a linear vector
// function P(t).
type Unit struct {
	ID        string
	OwnerSlot string
	Type      string
	HP        float64
	Origin    string
	Dest      string
	StartedAt time.Time
	ArrivesAt time.Time
	OriginX   float64
	OriginY   float64
	DestX     float64
	DestY     float64
	Speed     float64 // units / game-second
	Version   uint64  // bumped every time the unit accepts a new command

	Path      []PathLeg
	PathIndex int
	Waypoints []pathdom.Target
}

// PositionAt resolves the unit's position at the given game time using the
// linear motion equation from the architecture doc.
func (u *Unit) PositionAt(t time.Time) (float64, float64) {
	if len(u.Path) > 0 && u.PathIndex < len(u.Path) {
		leg := u.Path[u.PathIndex]
		if !t.After(u.StartedAt) {
			return leg.FromX, leg.FromY
		}
		if !t.Before(leg.ArrivesAt) {
			return leg.ToX, leg.ToY
		}
		totalDur := leg.ArrivesAt.Sub(u.StartedAt).Seconds()
		if totalDur <= 0 {
			return leg.ToX, leg.ToY
		}
		progress := t.Sub(u.StartedAt).Seconds() / totalDur
		return leg.FromX + (leg.ToX-leg.FromX)*progress,
			leg.FromY + (leg.ToY-leg.FromY)*progress
	}
	if u.Origin == u.Dest || u.ArrivesAt.IsZero() {
		return u.OriginX, u.OriginY
	}
	if !t.After(u.StartedAt) {
		return u.OriginX, u.OriginY
	}
	if !t.Before(u.ArrivesAt) {
		return u.DestX, u.DestY
	}
	totalDur := u.ArrivesAt.Sub(u.StartedAt).Seconds()
	if totalDur <= 0 {
		return u.DestX, u.DestY
	}
	progress := t.Sub(u.StartedAt).Seconds() / totalDur
	return u.OriginX + (u.DestX-u.OriginX)*progress,
		u.OriginY + (u.DestY-u.OriginY)*progress
}

// IsMoving reports whether the unit is in flight at the given game time.
func (u *Unit) IsMoving(t time.Time) bool {
	if len(u.Path) > 0 && u.PathIndex < len(u.Path) {
		return t.Before(u.Path[u.PathIndex].ArrivesAt)
	}
	return !u.ArrivesAt.IsZero() && t.Before(u.ArrivesAt) && u.Origin != u.Dest
}

// Province mirrors the static map definition with mutable owner state.
type Province struct {
	ID      string
	X       float64
	Y       float64
	Owner   string // slot id
	Capital bool
}

// Player is the runtime view of a slot. UserID may be empty before the
// slot is occupied (used in the lobby) and is set when the player joins.
type Player struct {
	Slot   string
	UserID string
	Name   string
	Color  string
	Alive  bool
}

// Building is a structure attached to a province. It mirrors the
// catalogue [buildingdom.Kind] but stores the per-instance owner.
type Building struct {
	Type      string `json:"type"`
	Province  string `json:"province"`
	OwnerSlot string `json:"owner_slot"`
}

// Match is the in-memory authoritative state for a single match. The
// engine keeps one of these per active match it owns. Mutations always go
// through the match-loop goroutine so we never need locks on the hot path.
type Match struct {
	ID          string
	MapID       string
	Map         *maps.Map
	Tick        uint64
	StartedAt   time.Time
	GameStart   time.Time
	GameNow     time.Time
	Speed       float64
	Players     map[string]*Player
	Provinces   map[string]*Province
	Units       map[string]*Unit
	Resources   map[string]*economydom.Resources
	Buildings   []*Building
	Timeline    *timeline.Timeline
	Status       string // active | ended
	WinnerSlot   string
	WinnerCoal   []string // every slot that shares the win (alliance victory)
	CleanupDone  bool     // set when the post-game cleanup event has fired
	Seq         uint64
	NextRecruit map[string]*RecruitOrder
	NextBuild   map[string]*BuildOrder
	Diplomacy   *diplomacydom.Registry

	// VisContributorVersion and VisContributors cache
	// [visibility.viewerContributors] results. Invalidated when diplomacy
	// changes (see [diplomacydom.Registry.Version]). Match-loop only.
	VisContributorVersion uint64
	VisContributors       map[string]map[string]bool
}

// SlotIDs returns every slot id present in the match, sorted. Used by
// diplomacy helpers that need a stable iteration order over slots.
func (m *Match) SlotIDs() []string {
	out := make([]string, 0, len(m.Players))
	for s := range m.Players {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// RecruitOrder is queued work to spawn a unit at a province after a delay.
type RecruitOrder struct {
	ID          string
	OwnerSlot   string
	Province    string
	UnitType    string
	IssuedAt    time.Time
	CompletesAt time.Time
}

// BuildOrder is queued construction of a building.
type BuildOrder struct {
	ID           string
	OwnerSlot    string
	Province     string
	BuildingType string
	IssuedAt     time.Time
	CompletesAt  time.Time
}

// HasBuilding reports whether a province already hosts a building of the
// given type.
func HasBuilding(m *Match, provinceID, buildingType string) bool {
	for _, b := range m.Buildings {
		if b.Province == provinceID && b.Type == buildingType {
			return true
		}
	}
	return false
}

// EuclidDistance helper for graph edges.
func EuclidDistance(ax, ay, bx, by float64) float64 {
	dx := ax - bx
	dy := ay - by
	return math.Sqrt(dx*dx + dy*dy)
}
