// Package ids defines the typed primitives for the IDs that flow through
// the simulation. Using distinct types instead of bare strings prevents
// the most common cross-wiring bugs (passing a UserID where a SlotID was
// expected, comparing a ProvinceID to a UnitID, etc.).
//
// Migration is gradual: as of Phase C the [cmddom.Command] inputs use
// these types; deeper adoption (Match.ID, Player.Slot) is tracked in the
// refactor backlog.
package ids
