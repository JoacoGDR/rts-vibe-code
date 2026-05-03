// Package log builds the structured logger used by every binary mode and
// provides helpers for ferrying it through context. All logs are JSON by
// default (configurable to text for local dev) with `service`, `mode` and
// `env` fields baked in by [New].
package log
