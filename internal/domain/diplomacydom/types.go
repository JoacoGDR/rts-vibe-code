package diplomacydom

import "time"

// Stance is the diplomatic posture between two slots. The default for any
// pair that has no explicit treaty is [War] — the simulation assumes
// hostile until proven otherwise, matching Supremacy's "everyone starts
// hostile until a treaty is signed" model.
type Stance string

const (
	War      Stance = "war"
	Peace    Stance = "peace"
	Alliance Stance = "alliance"
)

// Pending is the tri-state for a peace/alliance proposal that has been
// offered but not yet accepted by the other side. Peace and Alliance
// require both sides to opt in; War is always unilateral.
type Pending string

const (
	PendingNone     Pending = ""
	PendingPeace    Pending = "peace"
	PendingAlliance Pending = "alliance"
)

// Treaty is the pairwise diplomatic record. SlotA/SlotB are sorted
// alphabetically at construction so a single record represents both
// directions.
type Treaty struct {
	SlotA          string
	SlotB          string
	Stance         Stance
	Pending        Pending
	PendingFrom    string // the slot that proposed the Pending change
	ChangedAt      time.Time
	LastChangeFrom string // who initiated the most recent committed change
}

// Pact is a unilateral grant from one slot to another. Today the engine
// honours two kinds: ShareMap (vision) and RightOfWay (movement).
type Pact struct {
	From      string
	To        string
	Kind      PactKind
	GrantedAt time.Time
}

// PactKind enumerates the unilateral grants the engine understands.
type PactKind string

const (
	ShareMap   PactKind = "share_map"
	RightOfWay PactKind = "right_of_way"
)
