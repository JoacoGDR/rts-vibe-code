// Package wire defines the JSON message contract between the web client and
// the gateway, and between services on NATS. Keeping these types in a shared
// package enforces a single source of truth that compiles into every binary.
//
// JSON is the chosen wire format for MVP velocity. If bandwidth becomes a
// problem we can swap to Protobuf without changing the type names.
package wire

import "time"

// WireVersion is bumped when the JSON contract changes incompatibly.
// Clients should send it in hello and refuse mismatched servers.
const WireVersion = 3

type ClientMessageType string

const (
	ClientHello   ClientMessageType = "hello"
	ClientCommand ClientMessageType = "command"
	ClientPing    ClientMessageType = "ping"
	ClientResync  ClientMessageType = "resync"
	ClientChat    ClientMessageType = "chat"
	ClientGoodbye ClientMessageType = "goodbye"
)

type ServerMessageType string

const (
	ServerHelloAck ServerMessageType = "hello_ack"
	ServerState    ServerMessageType = "state"
	ServerEvent    ServerMessageType = "event"
	ServerChat     ServerMessageType = "chat"
	ServerError    ServerMessageType = "error"
	ServerPong     ServerMessageType = "pong"
)

type ClientEnvelope struct {
	Type        ClientMessageType `json:"type"`
	WireVersion int               `json:"wire_version,omitempty"`
	ID          string            `json:"id,omitempty"`
	MatchID     string            `json:"match_id,omitempty"`
	Command   *Command          `json:"command,omitempty"`
	Chat      *ChatOutbound     `json:"chat,omitempty"`
	ResyncSeq uint64            `json:"resync_seq,omitempty"`
}

type ServerEnvelope struct {
	Type    ServerMessageType `json:"type"`
	ID      string            `json:"id,omitempty"`
	MatchID string            `json:"match_id,omitempty"`
	Seq     uint64            `json:"seq,omitempty"`
	SentAt  time.Time         `json:"sent_at"`
	State   *MatchState       `json:"state,omitempty"`
	Event   *Event            `json:"event,omitempty"`
	Chat    *ChatInbound      `json:"chat,omitempty"`
	Error   *ErrorPayload     `json:"error,omitempty"`
}

// ChatOutbound is what a client sends when authoring a message. Scope is
// one of "world" / "coalition" / "dm". For DMs, TargetUserID identifies
// the recipient; for world / coalition it is ignored.
type ChatOutbound struct {
	Scope        string `json:"scope"`
	TargetUserID string `json:"target_user_id,omitempty"`
	Body         string `json:"body"`
}

// ChatInbound is what the server pushes back. Same shape as the
// persistence record; AuthorSlot may be empty when the author was a
// system / admin message.
type ChatInbound struct {
	ID         string    `json:"id"`
	MatchID    string    `json:"match_id"`
	Scope      string    `json:"scope"`
	AuthorID   string    `json:"author_user_id"`
	AuthorName string    `json:"author_display_name,omitempty"`
	AuthorSlot string    `json:"author_slot,omitempty"`
	Body       string    `json:"body"`
	SentAt     time.Time `json:"sent_at"`
}

type Command struct {
	Kind        string            `json:"kind"`
	UnitID      string            `json:"unit_id,omitempty"`
	From        string            `json:"from,omitempty"`
	To          string            `json:"to,omitempty"`
	IssuedAt    time.Time         `json:"issued_at"`
	Idempotency string            `json:"idempotency_key"`
	Args        map[string]string `json:"args,omitempty"`
}

type Event struct {
	Kind     string         `json:"kind"`
	OccurAt  time.Time      `json:"occur_at"`
	UnitID   string         `json:"unit_id,omitempty"`
	Province string         `json:"province,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MatchState struct {
	MatchID   string                  `json:"match_id"`
	Tick      uint64                  `json:"tick"`
	GameTime  time.Time               `json:"game_time"`
	Provinces []ProvinceState         `json:"provinces"`
	Units     []UnitState             `json:"units"`
	Players   []PlayerState           `json:"players"`
	Resources map[string]ResourcePool `json:"resources,omitempty"`
	Buildings []BuildingState         `json:"buildings,omitempty"`
	Queues    QueueState              `json:"queues,omitempty"`
	Diplomacy []TreatyState           `json:"diplomacy,omitempty"`
	Pacts     []PactState             `json:"pacts,omitempty"`
}

// TreatyState is the wire projection of one diplomatic record. Pending
// is empty when there is no outstanding offer.
type TreatyState struct {
	SlotA       string    `json:"slot_a"`
	SlotB       string    `json:"slot_b"`
	Stance      string    `json:"stance"`
	Pending     string    `json:"pending,omitempty"`
	PendingFrom string    `json:"pending_from,omitempty"`
	ChangedAt   time.Time `json:"changed_at"`
}

// PactState is the wire projection of one unilateral grant.
type PactState struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Kind      string    `json:"kind"`
	GrantedAt time.Time `json:"granted_at"`
}

type ResourcePool struct {
	Manpower float64 `json:"manpower"`
	Food     float64 `json:"food"`
	Iron     float64 `json:"iron"`
}

type BuildingState struct {
	Type     string `json:"type"`
	Province string `json:"province"`
	Owner    string `json:"owner"`
}

type QueueState struct {
	Recruits      []QueuedRecruit      `json:"recruits,omitempty"`
	Constructions []QueuedConstruction `json:"constructions,omitempty"`
}

type QueuedRecruit struct {
	Province    string    `json:"province"`
	UnitType    string    `json:"unit_type"`
	Owner       string    `json:"owner"`
	CompletesAt time.Time `json:"completes_at"`
}

type QueuedConstruction struct {
	Province     string    `json:"province"`
	BuildingType string    `json:"building_type"`
	Owner        string    `json:"owner"`
	CompletesAt  time.Time `json:"completes_at"`
}

type ProvinceState struct {
	ID      string  `json:"id"`
	OwnerID string  `json:"owner_id,omitempty"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Capital bool    `json:"capital,omitempty"`
}

type PathLegState struct {
	FromProv  string    `json:"from_prov,omitempty"`
	ToProv    string    `json:"to_prov,omitempty"`
	FromX     float64   `json:"from_x,omitempty"`
	FromY     float64   `json:"from_y,omitempty"`
	ToX       float64   `json:"to_x,omitempty"`
	ToY       float64   `json:"to_y,omitempty"`
	ArrivesAt time.Time `json:"arrives_at,omitempty"`
}

type UnitState struct {
	ID        string         `json:"id"`
	OwnerID   string         `json:"owner_id"`
	Type      string         `json:"type"`
	X         float64        `json:"x"`
	Y         float64        `json:"y"`
	HP        float64        `json:"hp"`
	Origin    string         `json:"origin,omitempty"`
	Dest      string         `json:"dest,omitempty"`
	StartedAt time.Time      `json:"started_at,omitempty"`
	ArrivesAt time.Time      `json:"arrives_at,omitempty"`
	Path      []PathLegState `json:"path,omitempty"`
	PathIndex int            `json:"path_index,omitempty"`
}

type PlayerState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	Alive bool   `json:"alive"`
}
