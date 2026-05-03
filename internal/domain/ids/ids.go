package ids

// MatchID identifies a single match instance.
type MatchID string

// SlotID identifies a player slot inside a match (e.g. "red", "blue").
// Distinct from [UserID] so the two never get accidentally swapped.
type SlotID string

// UserID identifies an authenticated user account.
type UserID string

// UnitID identifies an in-game unit.
type UnitID string

// ProvinceID identifies a node on the map graph.
type ProvinceID string

// BuildingID identifies a constructed building inside a province.
type BuildingID string

func (m MatchID) String() string    { return string(m) }
func (s SlotID) String() string     { return string(s) }
func (u UserID) String() string     { return string(u) }
func (u UnitID) String() string     { return string(u) }
func (p ProvinceID) String() string { return string(p) }
func (b BuildingID) String() string { return string(b) }
