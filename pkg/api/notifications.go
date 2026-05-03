package api

import (
	"time"

	"github.com/google/uuid"
)

// NotificationView is the public projection of a single notification —
// returned by GET /notifications and consumed by the bell UI.
type NotificationView struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	MatchID   uuid.UUID      `json:"match_id"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
	SentAt    *time.Time     `json:"sent_at,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
}

// ListNotificationsResponse is the body of GET /notifications.
// UnreadCount is included in the same response so the bell can refresh
// its badge in a single round-trip.
type ListNotificationsResponse struct {
	Items       []NotificationView `json:"items"`
	UnreadCount int                `json:"unread_count"`
}

// UnreadCountResponse is the body of GET /notifications/unread-count.
type UnreadCountResponse struct {
	Count int `json:"count"`
}

// MarkReadResponse echoes how many rows were touched so callers can
// short-circuit a bell refetch when the response is 0.
type MarkReadResponse struct {
	Marked int `json:"marked"`
}
