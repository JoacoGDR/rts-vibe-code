// Package visibility computes per-slot fog of war and produces the wire-
// shaped match snapshots the gateway broadcasts to clients. Both the
// authoritative ("public") snapshot and the per-slot filtered variant live
// here so the mapping logic stays in one place.
package visibility
