// Package platform contains the cross-cutting infrastructure helpers
// shared by every binary mode: structured logging, Prometheus metrics,
// the generic HTTP server runner with graceful shutdown, and the
// liveness/readiness probe surface.
//
// platform packages know nothing about game domain types. Adapters and
// the composition root (internal/app) pull what they need from here.
package platform
