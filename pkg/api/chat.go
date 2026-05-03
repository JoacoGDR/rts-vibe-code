package api

import (
	"time"

	"github.com/google/uuid"
)

// SendChatRequest is the body of POST /matches/{matchID}/chat. Scope is
// the same canonical key the WebSocket protocol uses.
type SendChatRequest struct {
	Scope string `json:"scope"`
	Body  string `json:"body"`
	// AuthorSlot is filled in by the client (or omitted, in which case
	// the controller looks it up from the match roster).
	AuthorSlot string `json:"author_slot,omitempty"`
}

// ChatMessageView is the public projection of a single chat message —
// used by both POST /chat (echo) and GET /chat (history).
type ChatMessageView struct {
	ID         uuid.UUID `json:"id"`
	MatchID    uuid.UUID `json:"match_id"`
	Scope      string    `json:"scope"`
	AuthorID   uuid.UUID `json:"author_user_id"`
	AuthorSlot string    `json:"author_slot"`
	Body       string    `json:"body"`
	SentAt     time.Time `json:"sent_at"`
}
