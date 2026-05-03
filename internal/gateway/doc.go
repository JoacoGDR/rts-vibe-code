// Package gateway is the composition root for the gateway binary mode.
// It terminates client WebSocket connections, exchanges WS tickets for
// sessions, validates commands and bridges JSON payloads onto the NATS
// subjects the engine consumes via [internal/adapter/natsbridge].
package gateway
