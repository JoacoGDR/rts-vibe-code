// Package balance is the single source of truth for tunable game numbers:
// starting resources, view radii, macro pulse cadence, default unit speed,
// and any other constant a designer might want to nudge between releases.
// Keeping them in one place means rebalancing patches read like a diff
// against a small file instead of a hunt across the codebase.
package balance
