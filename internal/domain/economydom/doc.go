// Package economydom owns the resource value object used everywhere costs,
// stockpiles or production flow through the simulation. Resources is a
// pure value type with arithmetic helpers and no hidden state — every
// member is exported and mutation happens by explicit Add/Pay calls.
package economydom
