// Package buildingdom is the building-type catalogue. Same Strategy
// pattern as [unitdom]: each building kind is a Go type implementing
// [Kind]; registry.go maps string ids to instances. Production() lets the
// macro-pulse computation ask "what does this building add to the
// stockpile?" without ever switching on a type string.
package buildingdom
