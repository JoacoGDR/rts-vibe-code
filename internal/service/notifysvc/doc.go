// Package notifysvc owns the notification use cases that core-api and
// the worker share: insertion (with debounce), per-user listing and
// mark-as-read. The package is intentionally read-heavy on the core-api
// side; the worker also constructs a Service so it can call Insert with
// the same debounce policy whether the trigger came from a NATS event
// or from a one-off API call (handoff, AI takeover).
package notifysvc
