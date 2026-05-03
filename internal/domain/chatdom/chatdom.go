package chatdom

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// ChannelKind enumerates the high-level chat channels the engine knows.
type ChannelKind string

const (
	ChannelWorld     ChannelKind = "world"
	ChannelCoalition ChannelKind = "coalition"
	ChannelDM        ChannelKind = "dm"
)

// MaxBodyBytes is the hard cap on a single chat message. Larger writes
// are rejected by the service with [ErrBodyTooLong].
const MaxBodyBytes = 1024

// Sentinel validation errors.
var (
	ErrEmptyBody     = errors.New("chat body must not be empty")
	ErrBodyTooLong   = errors.New("chat body exceeds size cap")
	ErrControlChars  = errors.New("chat body contains control characters")
	ErrUnknownScope  = errors.New("unknown chat scope")
	ErrNotInScope    = errors.New("user is not allowed to post in scope")
	ErrInvalidTarget = errors.New("invalid dm target")
)

// Message is the persisted chat record. ID is allocated by the service
// (UUID v4); the storage layer treats it as a primary key.
type Message struct {
	ID         uuid.UUID
	MatchID    uuid.UUID
	Scope      string
	AuthorID   uuid.UUID
	AuthorSlot string
	Body       string
	SentAt     time.Time
}

// HistoryQuery is the parameter bag for paged history reads. Lives in
// the domain so both the service and the persistence layer can consume
// the same shape without one depending on the other.
type HistoryQuery struct {
	MatchID uuid.UUID
	Scope   string
	Before  time.Time
	Limit   int
}

// WorldScope is the canonical scope key for the match-wide channel.
func WorldScope(matchID uuid.UUID) string { return fmt.Sprintf("world:%s", matchID) }

// CoalitionScope is the canonical key for an alliance channel. The
// coalition leader id (the alphabetically smallest slot id in the
// coalition) is supplied by the caller — keeps this package free of
// diplomacy imports.
func CoalitionScope(matchID uuid.UUID, coalitionID string) string {
	return fmt.Sprintf("coal:%s:%s", matchID, coalitionID)
}

// DMScope is the canonical key for a 1-1 chat between two users. The two
// user IDs are sorted so a/b and b/a resolve to the same record.
func DMScope(matchID, a, b uuid.UUID) string {
	x, y := a.String(), b.String()
	pair := []string{x, y}
	sort.Strings(pair)
	return fmt.Sprintf("dm:%s:%s:%s", matchID, pair[0], pair[1])
}

// ParseScope dissects a scope key into its kind and trailing parts. Used
// by chatsvc.Send to decide which membership check applies.
func ParseScope(scope string) (ChannelKind, []string, error) {
	parts := strings.Split(scope, ":")
	if len(parts) < 2 {
		return "", nil, ErrUnknownScope
	}
	switch parts[0] {
	case "world":
		if len(parts) != 2 {
			return "", nil, ErrUnknownScope
		}
		return ChannelWorld, parts[1:], nil
	case "coal":
		if len(parts) != 3 {
			return "", nil, ErrUnknownScope
		}
		return ChannelCoalition, parts[1:], nil
	case "dm":
		if len(parts) != 4 {
			return "", nil, ErrUnknownScope
		}
		return ChannelDM, parts[1:], nil
	default:
		return "", nil, ErrUnknownScope
	}
}

// ValidateBody trims the body and checks size + control characters. The
// returned string is the value the caller should persist.
func ValidateBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", ErrEmptyBody
	}
	if len(body) > MaxBodyBytes {
		return "", ErrBodyTooLong
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrControlChars
		}
	}
	return body, nil
}
