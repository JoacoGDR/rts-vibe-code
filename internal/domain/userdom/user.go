package userdom

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User is the persisted account record. IsBot is set on pre-seeded
// service accounts that the AI bot subsystem (`ai-bot` mode) uses to
// authenticate as a real WebSocket client; the LoginAsBot endpoint
// rejects every account where IsBot is false.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
	Color        string
	IsBot        bool
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// Sentinel errors that the authsvc and pgrepo layers swap through. Stable
// across implementations so HTTP controllers can map them to status codes.
var (
	ErrNotFound = errors.New("user not found")
	ErrExists   = errors.New("user already exists")
)
