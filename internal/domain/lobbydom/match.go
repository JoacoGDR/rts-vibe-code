package lobbydom

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Match is the lobby/persistence record. The engine has its own in-memory
// [matchdom.Match] aggregate; this type is what core-api hands the
// frontend before (and during) a session.
type Match struct {
	ID           uuid.UUID
	Name         string
	MapID        string
	Status       Status
	CreatedBy    uuid.UUID
	CreatedAt    time.Time
	StartedAt    *time.Time
	EndedAt      *time.Time
	WinnerUserID *uuid.UUID
	SpeedFactor  float64
}

// Player is a single seat in a match: a user who has joined a slot.
// ControlledByAI flips when the AI takeover threshold is hit (or the
// player explicitly hands control over) and tells the rest of the stack
// that commands for this slot now come from the ai-bot binary mode.
// LastSeenAt is the wall-clock timestamp of the most recent heartbeat
// the worker observed; nil means "never connected".
type Player struct {
	MatchID        uuid.UUID
	UserID         uuid.UUID
	Slot           string
	Color          string
	Alive          bool
	ControlledByAI bool
	LastSeenAt     *time.Time
	JoinedAt       time.Time
}

// Sentinel errors lifted out of the persistence layer.
var (
	ErrNotFound         = errors.New("match not found")
	ErrAlreadyRunning   = errors.New("match already started")
	ErrNotEnoughPlayers = errors.New("match needs at least 2 players")
	ErrSlotTaken        = errors.New("slot is already taken")
	ErrNoFreeSlot          = errors.New("no free slot available")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrNotCreator          = errors.New("only the match creator may perform this action")
	ErrCannotLeaveActive   = errors.New("use handoff to leave an active match")
	ErrPlayerNotInMatch    = errors.New("player is not in this match")
)
