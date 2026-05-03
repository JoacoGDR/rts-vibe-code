// Package natsbridge is the single source of truth for everything the
// engine sends or receives over NATS. It owns the subject naming scheme,
// the JSON payload types (CommandPayload, StartPayload), the JetStream
// stream/consumer setup, the publisher helpers and the subscriber
// callbacks. Both gateway and engine import this package — neither needs
// to know about the other.
package natsbridge
