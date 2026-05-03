// Package matchdom is the heart of the simulation: the Match aggregate,
// its sub-entities (Unit, Player, Province, Building, Resources) and the
// pure-Go logic that mutates them in response to commands and timeline
// events. No infrastructure imports — everything here is testable with
// only stdlib + pkg/.
package matchdom
