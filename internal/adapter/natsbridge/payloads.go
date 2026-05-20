package natsbridge

import "time"

// CommandPayload is the on-wire shape of a command flowing through NATS
// from the gateway to the engine. The gateway has already authenticated
// and tagged the issuer.
type CommandPayload struct {
	MatchID        string            `json:"match_id"`
	UserID         string            `json:"user_id"`
	Slot           string            `json:"slot"`
	IdempotencyKey string            `json:"idempotency_key"`
	Kind           string            `json:"kind"`
	UnitID         string            `json:"unit_id,omitempty"`
	From           string            `json:"from,omitempty"`
	To             string            `json:"to,omitempty"`
	IssuedAt       time.Time         `json:"issued_at"`
	Args           map[string]string `json:"args,omitempty"`
}

// StartPayload is what core-api publishes to ask the engine to materialise a
// new match.
type StartPayload struct {
	MatchID         string            `json:"match_id"`
	MapID           string            `json:"map_id"`
	Speed           float64           `json:"speed"`
	StartedAt       time.Time         `json:"started_at"`
	SlotAssignments map[string]string `json:"slot_assignments"`
}

// ResyncPayload is the body of a resync request fired by the gateway.
type ResyncPayload struct {
	MatchID string `json:"match_id"`
}

// EndMatchPayload asks the engine to tear down a hosted match (used by
// the abandonment worker when every human has gone silent).
type EndMatchPayload struct {
	MatchID string `json:"match_id"`
	Reason  string `json:"reason,omitempty"`
}
