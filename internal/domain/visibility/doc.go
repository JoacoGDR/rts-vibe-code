// Package visibility computes per-slot fog of war and produces the wire-
// shaped match snapshots the gateway broadcasts to clients. Both the
// authoritative ("public") snapshot and the per-slot filtered variant live
// here so the mapping logic stays in one place.
//
// For engine publish cycles that fan out to many slots, use [Batch] so
// [For] runs once per slot per instant and shared wire slices are built once.
// Circle–point tests use a uniform spatial index; diplomacy contributor sets
// are cached on the match until [diplomacydom.Registry.Version] changes.
package visibility
