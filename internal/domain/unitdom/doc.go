// Package unitdom is the unit-type catalogue. Each unit kind is a Go type
// implementing [Kind]; adding a new unit is exactly one new file plus one
// line in registry.go.
//
// This package replaces the switch-on-string `unitStats` previously living
// in `engine/economy.go` and the parallel `damageMatrix` table.
package unitdom
