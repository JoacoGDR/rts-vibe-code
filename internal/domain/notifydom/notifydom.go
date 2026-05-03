// Package notifydom is the domain layer for the per-user notifications
// the worker emits on match events. The persistence layer lives in
// `pgrepo.Notifications`; the worker pipeline (subscribe → debounce →
// persist → email) lives in `internal/worker/notify.go`; the bell endpoint
// lives in `internal/adapter/httpapi/v1/notifyctl.go`.
//
// The package owns three things:
//
//   - The Kind enum, so we have a single source of truth for the wire
//     identifiers core-api, the worker and the SPA all share.
//   - The Notification persisted record.
//   - DebounceWindow, the per-(user, kind) interval the worker enforces
//     so a stream of `province_captured` events doesn't drown the inbox.
package notifydom

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Kind enumerates the notification types the worker recognises today.
// Every kind has a corresponding template in `internal/worker/notify.go`
// that builds the payload + subject/body strings.
type Kind string

const (
	KindProvinceCaptured Kind = "province_captured"
	KindUnderAttack      Kind = "under_attack"
	KindTreatyProposed   Kind = "treaty_proposed"
	KindMatchEnded       Kind = "match_ended"
	KindBotTakeover      Kind = "bot_takeover"
)

// DebounceWindow returns the minimum spacing the worker enforces between
// two notifications of the same Kind for the same user/match. Returning
// 0 means "never debounce" (used for one-shot events like match_ended).
func (k Kind) DebounceWindow() time.Duration {
	switch k {
	case KindUnderAttack, KindProvinceCaptured:
		return 5 * time.Minute
	case KindTreatyProposed, KindBotTakeover:
		return 30 * time.Second
	case KindMatchEnded:
		return 0
	default:
		return time.Minute
	}
}

// Notification is the persisted record. SentAt is nil while the email
// is still queued; ReadAt is nil while the user has not opened it in
// the bell UI yet.
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	MatchID   uuid.UUID
	Kind      Kind
	Payload   map[string]any
	CreatedAt time.Time
	SentAt    *time.Time
	ReadAt    *time.Time
}

// ListQuery is the parameter bag for the bell endpoint's pagination.
type ListQuery struct {
	UserID uuid.UUID
	After  time.Time // exclusive lower bound on CreatedAt
	Limit  int
}

// ErrNotFound is returned by the persistence layer when a single-row
// lookup misses.
var ErrNotFound = errors.New("notification not found")
