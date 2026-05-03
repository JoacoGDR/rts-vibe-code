package api

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest is the body of POST /auth/register.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is returned on successful register/login.
type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        UserView  `json:"user"`
}

// WSTicketResponse holds a single-use ticket that can be exchanged for
// a WebSocket session against the gateway.
type WSTicketResponse struct {
	Ticket string `json:"ticket"`
}

// UserView is the public projection of a user account.
type UserView struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Color       string    `json:"color"`
}

// MeResponse is what GET /auth/me returns.
type MeResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// LoginAsBotRequest is the body of POST /auth/login-as-bot. Only the
// ai-bot subsystem ever calls this — the controller checks the
// X-Bot-API-Key header against the BOT_API_KEY config before invoking
// the service.
type LoginAsBotRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// BotUserView is the read projection ai-bot consumes when discovering
// available bot identities.
type BotUserView struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
}

// ListBotsResponse is the body of GET /auth/bots.
type ListBotsResponse struct {
	Bots []BotUserView `json:"bots"`
}
