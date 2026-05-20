// Package app is the composition root for every binary mode. It is the
// only place where concrete adapters (Postgres, Redis, NATS) get wired
// into the services that the controllers depend on. Everything below
// service/ depends on interfaces declared by their consumers, so app/
// is the single layer that "knows" the whole graph.
//
// Production runs via cmd/core-api; this package is the core-api
// composition root. One file per mode keeps the wiring shallow and
// grep-able. Adding a
// new dependency means editing exactly one file.
package app
