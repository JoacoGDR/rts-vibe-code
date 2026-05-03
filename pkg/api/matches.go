package api

import (
	"time"

	"github.com/google/uuid"
)

// CreateMatchRequest is the body of POST /matches.
type CreateMatchRequest struct {
	Name  string `json:"name"`
	MapID string `json:"map_id"`
	Slot  string `json:"slot"`
}

// JoinMatchRequest is the body of POST /matches/{id}/join.
type JoinMatchRequest struct {
	Slot string `json:"slot"`
}

// MatchView is the public projection of a single match.
type MatchView struct {
	ID        uuid.UUID    `json:"id"`
	Name      string       `json:"name"`
	MapID     string       `json:"map_id"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	StartedAt *time.Time   `json:"started_at,omitempty"`
	EndedAt   *time.Time   `json:"ended_at,omitempty"`
	WinnerID  *uuid.UUID   `json:"winner_user_id,omitempty"`
	Players   []PlayerView `json:"players"`
}

// PlayerView is the public projection of a match player. ControlledByAI
// is true once the heartbeat-based takeover scanner has flipped the
// slot or the player explicitly handed control to the bot.
type PlayerView struct {
	UserID         uuid.UUID `json:"user_id"`
	Slot           string    `json:"slot"`
	Color          string    `json:"color"`
	Alive          bool      `json:"alive"`
	ControlledByAI bool      `json:"controlled_by_ai,omitempty"`
}

// JoinMatchResponse is the body of a successful join. When the user was
// already in the match the Status is "already_joined" and Slot reflects
// their existing slot.
type JoinMatchResponse struct {
	Status string `json:"status"`
	Slot   string `json:"slot"`
}
