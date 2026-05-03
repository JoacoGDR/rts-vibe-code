// Package lobbydom owns the persistent lobby record: matches that exist in
// Postgres but may not be running on any engine instance yet. Distinct
// from [matchdom], which is the in-memory simulation aggregate.
package lobbydom
