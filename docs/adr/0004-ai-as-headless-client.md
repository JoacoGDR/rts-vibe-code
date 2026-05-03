# ADR-0004: AI Bot as a Headless WebSocket Client

- **Status**: accepted
- **Date**: 2026-05-02
- **Deciders**: @joaquing
- **Phase**: Phase 5 (AI Takeover & Notifications)

## Context

Phase 5 introduces a heuristic AI that takes over a slot when its
human player has been silent for too long. The constraints are:

1. **Realism.** A bot must be indistinguishable from a player on the
   wire so the engine, gateway and web client need zero special cases.
2. **Trust boundary.** Bots are still principals — they must
   authenticate, they must obey rate limits, they must not bypass
   command validation.
3. **Single binary, microservice tomorrow.** The bot subsystem must
   survive being moved into its own deployment unit (already true via
   Docker Compose).
4. **Pluggable strategy.** The decision logic should be pure Go so it
   can be unit-tested against fixture states without spinning up
   Postgres/NATS.

Two alternatives were considered up-front:

- **In-engine bots.** Spawn a goroutine inside the engine that
  synthesises commands directly into the match loop. Rejected — the
  engine would need to know about diplomacy, chat, and rate limits;
  any bot bug would crash a real match's goroutine.
- **NATS-only "headless" bots.** Have the bot publish on
  `match.<id>.cmd` directly, skipping the gateway. Rejected — bypasses
  the auth layer and the per-user rate limiter, and the bot wouldn't
  see the per-slot filtered state.

## Decision

The `ai-bot` binary mode is a **real WebSocket client**. The flow is:

1. **Discovery.** A new Postgres column `match_players.controlled_by_ai`
   gets flipped to `true` by the worker's presence scanner once the
   user has not opened a WebSocket session for the match in
   `AI_TAKEOVER_AFTER` (default 72h). Presence is recorded by a single
   `match.<id>.presence.<slot>` NATS message the gateway publishes on
   each WebSocket hello — there is no continuous heartbeat. The
   scanner runs once per `AI_POLL_INTERVAL` (default 1h).
2. **Pool.** The `ai-bot` runner polls `match_players` for
   `controlled_by_ai = TRUE` rows. For each new row it picks a
   service-account from the seed pool of `users.is_bot = TRUE`
   identities (round-robin) and spawns a `botSession` goroutine.
3. **Auth.** The session calls
   `POST /api/v1/auth/login-as-bot`, guarded by the pre-shared
   `BOT_API_KEY` header (`mw.RequireBotKey`). The endpoint refuses any
   account where `is_bot = false`.
4. **Ticket.** The session exchanges its JWT for a one-shot ticket
   via the existing `POST /api/v1/auth/ws-ticket`.
5. **Connect.** The session dials `ws://gateway:8081/ws`, sends a
   `hello`, and from there speaks the same wire protocol the React
   client uses — `ClientCommand`, `ClientPing`, `ClientResync`,
   `ClientGoodbye`.
6. **Decide.** Every state envelope is fed to
   `internal/aibot/heuristic.Decide(state, ownSlot, prng)`. The
   heuristic is pure Go with no I/O, so it can be unit-tested against
   fixture states (see `heuristic_test.go`).

The pieces:

- **`internal/aibot/coreclient.go`** — typed HTTP wrapper for
  `ListBots`, `LoginAsBot`, `MintWSTicket`. Authenticates using either
  `X-Bot-API-Key` or `Authorization: Bearer <jwt>` per call.
- **`internal/aibot/session.go`** — owns one WebSocket. Throttles
  decisions to once per ~750 ms so a state burst doesn't translate
  into a command burst.
- **`internal/aibot/runner.go`** — the binary mode's main loop:
  refresh the bot pool, scan for AI-controlled slots, reconcile in-
  memory sessions.
- **`internal/aibot/heuristic`** — pure-Go decision functions: defend
  capital, recruit if affordable, dispatch idle units, accept peace
  offers. Hard-capped at `MaxCommandsPerDecision = 6` so a buggy
  heuristic cannot drown the gateway.

## Consequences

- **Positive**:
  - Bot commands hit the same auth, rate limiting and validation as
    real players. A new command kind needs zero changes in `ai-bot`.
  - The bot subsystem can be horizontally scaled by running multiple
    `ai-bot` processes — each one only takes the (match, slot) tuples
    no peer is already serving (today via simple per-process state;
    Phase 6 will add a Redis lock if we go multi-instance).
  - The heuristic is testable. The 8 tests in `heuristic_test.go`
    cover: no-state, recruit-when-affordable, no-recruit-when-broke,
    attack-closest-hostile, respect-peace, accept-offered-peace,
    no-double-recruit, unique-idempotency.
- **Negative**:
  - Each bot session opens a real WebSocket — adds load on the gateway
    proportional to the number of unattended slots. We accept this
    because we expect ≤8 unattended slots per match in the MVP.
  - The bot pool is configured by a SQL `INSERT … ON CONFLICT DO
    NOTHING` in the migration. A future operator may want to add or
    rename bots; until then the seed pool is fixed at six identities.
  - Login round-trip per bot adds latency before commands flow. For a
    72h idle threshold this is invisible.
  - Presence is connect-only: a user who stays connected for days but
    never re-opens the page does not get a fresh ping until they
    reconnect. We accept this — they are still actively connected, and
    the gateway will close idle sockets independently via `pongWait`,
    which forces a reconnect (and a fresh presence ping) anyway.
- **Follow-ups**:
  - Phase 6 will add a Redis claim so multiple `ai-bot` processes can
    share the takeover load without racing on the same slot.
  - The heuristic is intentionally simple. A future ADR should cover
    swapping in MCTS or a learned policy if the gameplay needs it.
  - A "Hand to AI" button (Phase 6) flips `controlled_by_ai = true`
    immediately, reusing the same session loop without changes.

## Alternatives considered

- **In-engine bots.** Rejected (above).
- **NATS-only headless bots.** Rejected (above).
- **gRPC sidecar.** Considered: bots could speak protobuf to a thin
  gRPC server inside the engine. Rejected for MVP — duplicates the WS
  path's auth layer and adds a new transport.
- **HTTP polling.** Considered: bots could poll the state via REST.
  Rejected — doesn't surface events fast enough for combat reactions
  and would need a new "events since seq" endpoint.
