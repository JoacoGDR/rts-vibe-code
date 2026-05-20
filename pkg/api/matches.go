package api

import (
	"time"

	"github.com/google/uuid"
)

// CreateMatchRequest is the body of POST /matches.
type CreateMatchRequest struct {
	Name      string `json:"name"`
	MapID     string `json:"map_id"`
	Slot      string `json:"slot"`
	AutoStart *bool  `json:"auto_start,omitempty"` // nil/true = fill AI and start immediately
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

// KickMatchRequest is the body of POST /matches/{id}/kick.
type KickMatchRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// PatchSlotRequest toggles AI fill on an empty waiting slot.
type PatchSlotRequest struct {
	Slot     string `json:"slot"`
	EnableAI bool   `json:"enable_ai"`
}

// MatchStatsView is the post-game summary from GET /matches/{id}/stats.
type MatchStatsView struct {
	MatchID       uuid.UUID          `json:"match_id"`
	DurationSec   int                `json:"duration_sec"`
	TotalUnits    int                `json:"total_units"`
	TotalCombats  int                `json:"total_combats"`
	CapitalsTaken int                `json:"capitals_taken"`
	PerSlot       map[string]any     `json:"per_slot"`
}
