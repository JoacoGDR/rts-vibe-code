// Package pgrepo holds the Postgres-backed repository adapters: users,
// matches, snapshots. Each repository is one small file with a constructor
// that takes a *pgxpool.Pool, methods that mirror an interface declared in
// the consuming service package, and SQL kept inline so the structure
// query is one screen away from the Go code that runs it.
package pgrepo
