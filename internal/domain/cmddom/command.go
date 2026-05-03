package cmddom

import (
	"errors"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/ids"
)

// Command is the engine's internal representation of a player order. The
// gateway translates incoming wire messages into Commands before
// publishing them to NATS, and the engine applies them to the match.
//
// Args carries command-specific string params that don't fit cleanly into
// the canonical From/To/UnitID slots — for example "type=infantry" for
// recruit, or "type=factory" for construct.
type Command struct {
	MatchID        ids.MatchID
	UserID         ids.UserID
	IssuerSlot     ids.SlotID
	IdempotencyKey string
	Kind           string
	UnitID         ids.UnitID
	From           ids.ProvinceID
	To             ids.ProvinceID
	IssuedAt       time.Time
	Args           map[string]string
}

// Extra is a typed accessor for Args. Returns ("", false) when absent.
func (c Command) Extra(key string) (string, bool) {
	if c.Args == nil {
		return "", false
	}
	v, ok := c.Args[key]
	return v, ok
}

// Sentinel errors used by handlers. Group them here so callers can switch
// on errors.Is without importing each handler file.
var (
	ErrUnknownUnit       = errors.New("unknown unit")
	ErrUnauthorized      = errors.New("unit does not belong to issuer")
	ErrInvalidDest       = errors.New("destination is not adjacent")
	ErrUnknownCommand    = errors.New("unknown command kind")
	ErrUnitAlreadyMoving = errors.New("unit is already moving")
	ErrInsufficientFunds = errors.New("not enough resources")
	ErrInvalidProvince   = errors.New("invalid province")
	ErrUnknownUnitType   = errors.New("unknown unit type")
	ErrFactoryRequired   = errors.New("a factory is required to build this unit")
	ErrAlreadyQueued     = errors.New("province already has work in progress")
	ErrMatchInactive     = errors.New("match not active")
	ErrUnknownTarget     = errors.New("unknown target slot")
	ErrDiplomacy         = errors.New("diplomacy action rejected")
)
