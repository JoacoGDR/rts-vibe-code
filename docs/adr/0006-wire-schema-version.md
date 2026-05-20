# ADR 0006: Wire schema versioning

## Status

Accepted

## Context

Phases 4–6 extend the WebSocket JSON contract (diplomacy state, chat, coalition
victory payloads). Clients and servers must agree on incompatible field shapes.

## Decision

- Add `wire.WireVersion = 2` in `pkg/shared/wire/messages.go`.
- Clients send `wire_version` on `hello`; the gateway rejects mismatched values.
- Bump the constant and this ADR when making breaking wire changes.

## Consequences

- Older clients without `wire_version` are accepted (zero means unspecified).
- New clients can fail fast instead of mis-parsing state envelopes.
