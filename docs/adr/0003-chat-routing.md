# ADR-0003: Chat Routing — Gateway-Trusted Membership, Core-API Persistence

- **Status**: accepted
- **Date**: 2026-05-02
- **Deciders**: @joaquing
- **Phase**: Phase 4 (Diplomacy & Chat)

## Context

Phase 4 introduces in-match chat with three scopes — `world`, `coalition`,
and `dm`. The constraints are:

1. **Coalition membership is dynamic.** A user may join or leave a
   coalition at any time as alliances shift. Authorisation has to follow
   live diplomacy state, not a static roster.
2. **Persistence must be transactional with simulation lookups.** The UI
   re-loads recent history on reconnect, so messages need a durable home
   (Postgres) on the same hop as the validation.
3. **Live fan-out should be as cheap as possible.** Chat traffic is
   small but it sits next to the much hotter state stream and must not
   slow the engine.
4. **Single binary today, microservice tomorrow.** Routing decisions
   must survive moving the engine, gateway and core-api into separate
   processes (already true via Docker Compose).

The naive "engine owns chat too" routing was rejected immediately: it
would force every chat message through the simulation goroutine and
couple chat throttling to the tick rate.

## Decision

Chat is owned end-to-end by **core-api**. The gateway is a smart
forwarder that owns membership decisions and per-user fan-out
subscriptions. Concretely:

- **chatdom** owns the canonical scope grammar (`world:<matchID>`,
  `coal:<matchID>:<coalitionLeader>`, `dm:<matchID>:<userA>:<userB>`),
  body validation, and `chatdom.Message`.
- **chatsvc** in `internal/service/chatsvc` is the use case: validate
  body + scope, check that the author is in the match, throttle, persist
  via `pgrepo.Chat`, then publish to NATS via the chat scope subject.
- **pgrepo.Chat** persists into a single `chat_messages(scope, sent_at)`
  table indexed for newest-first paging.
- **natsbridge.ChatIngressSubject** (`chat.<matchID>.ingress`) is what
  the gateway publishes to when a user authors a message; core-api
  subscribes to `chat.*.ingress`. Outbound fan-out subjects live under
  `chat.scope.*` so the gateway can use NATS wildcards for coalition
  filtering and per-user DM inboxes.
- **Gateway responsibilities** (`internal/gateway/chat.go`):
  - Subscribe to world, dm-inbox-for-self, and coalition wildcard for
    the connected match on hello.
  - Track the user's current coalition leader by peeking at the
    diplomacy section of every state envelope; filter coalition
    messages locally so only the current coalition's traffic reaches
    the WS.
  - Translate the user-facing scope (`"world" / "coalition" / "dm"`)
    into a fully qualified scope key when forwarding to core-api.
- **Coalition trust boundary**: the gateway is the source of truth for
  which coalition the author is currently in. Chatsvc only validates
  match membership and DM participant identity; coalition membership is
  not re-derived on the core-api side because the engine state lives
  elsewhere. A misbehaving gateway can only post to coalitions it picks
  for itself, and only members of that coalition will ever subscribe to
  the corresponding subject — no information leaks across coalitions.
- **DM fan-out**: the service publishes one message per participant on a
  per-user inbox subject (`dm:<matchID>:inbox:<userID>`). The gateway
  subscribes to its own inbox once and gets every conversation it is in.

## Consequences

- **Positive**:
  - Chat traffic stays off the engine goroutine; throttling and writes
    happen in core-api which already owns Postgres.
  - The wire format is symmetrical (`scope`, `body`) for WS and REST,
    so the REST endpoint can be used by curl, integration tests and
    chat-only clients without a WebSocket.
  - Adding a new scope kind (e.g. `team:<id>`) is a one-file change in
    `chatdom.ParseScope` and a corresponding gateway subscription
    update.
- **Negative**:
  - The gateway must parse every state envelope to track coalition
    membership. The cost is bounded (≤8 slots, O(N²) edges) but it
    runs on the hot fan-out path.
  - Coalition membership trust is split across two processes; a
    privileged client could spam its own coalition's channel with
    forged scope keys until banned. We accept this for MVP because the
    blast radius is one coalition.
- **Follow-ups**:
  - Per-coalition rate limits in addition to per-user (Phase 7).
  - Move the trust boundary into the engine if cross-coalition
    inference becomes a concern.
  - Add `system:` scope for engine-emitted notifications (war
    declarations, ceasefires) once Phase 5's notification table lands.

## Alternatives considered

- **Engine owns chat.** Rejected — couples chat to tick rate and forces
  Postgres writes onto the simulation goroutine.
- **Gateway owns persistence.** Rejected — the gateway already does
  enough (auth, rate limiting, fan-out); adding a Postgres dependency
  there increases its blast radius and breaks the "gateway is stateless"
  constraint.
- **Two-phase auth: chatsvc re-derives coalition by querying the engine
  via NATS request/reply.** Rejected for MVP — adds a synchronous hop
  to every chat send and re-implements diplomacy lookup outside the
  engine. Revisit in Phase 7 if we adopt protobuf and an internal RPC
  channel.
