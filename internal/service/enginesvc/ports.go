package enginesvc

import "context"

// Broadcaster is the port the runner uses to push state and event payloads
// onto whatever transport is wired in. The natsbridge adapter is the only
// implementation today. An empty `slot` means "publish on the public/un-
// filtered subject for that match".
type Broadcaster interface {
	PublishPublicState(ctx context.Context, matchID string, payload []byte) error
	PublishSlotState(ctx context.Context, matchID, slot string, payload []byte) error
	PublishPublicEvent(ctx context.Context, matchID string, payload []byte) error
	PublishSlotEvent(ctx context.Context, matchID, slot string, payload []byte) error
}

// SlotIndex is implemented by anything that can persist the user->slot
// mapping for a match (Redis in production, in-memory for tests). The
// engine writes to it when a match starts so the gateway can resolve the
// right per-slot subject without a database round trip.
type SlotIndex interface {
	SetUserSlot(matchID, userID, slot string) error
}

type noopSlotIndex struct{}

func (noopSlotIndex) SetUserSlot(string, string, string) error { return nil }
