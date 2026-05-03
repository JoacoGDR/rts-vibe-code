// Package storage owns the connection helpers for Postgres, Redis and
// NATS plus the goose migrations bundle. It does not know about domain
// types — every consumer pulls a typed adapter (pgrepo, redisrepo,
// natsbridge) on top of these clients.
package storage
