// Package redisrepo holds the Redis-backed adapters: the per-match slot
// index that maps user IDs to slot IDs, and (in later phases) idempotency
// caches and the WS ticket store. Each repo is one small file.
package redisrepo
