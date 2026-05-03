# ADR-0001: Layered Modular Monolith with Hex-Flavoured Adapters

- **Status**: accepted
- **Date**: 2026-05-01
- **Deciders**: @joaquing
- **Phase**: code-quality refactor (between Phase 3 and Phase 4 of the
  product roadmap)

## Context

Phases 0–3 of the product roadmap shipped a working monolith with all
backend code under `internal/coreapi`, `internal/engine`,
`internal/gateway` and `internal/worker`. By the end of Phase 3 several
smells were getting hard to ignore:

- `internal/engine/economy.go` had three switch-on-string functions
  (`unitStats`, `damageMatrix`, `buildingCost`) — adding a unit type was
  five edits across the file.
- `internal/coreapi/handlers.go` was a single 400-line file mixing HTTP
  parsing, business logic, persistence and response shaping.
- The gateway imported the engine package directly for command payloads,
  coupling two binary modes that should only share a wire contract.
- HTTP server boilerplate, CORS middleware and error handling were
  duplicated across every `Run()` function.
- Domain ids (`MatchID`, `UserID`, `SlotID`) were bare strings and got
  swapped at call sites.
- Frontend `MatchView.tsx` mixed data fetch, socket lifecycle, command
  dispatch and rendering.

Phase 4 (diplomacy / chat / AI takeover) would have made all of these
strictly worse. We paused feature work and refactored.

The user's stated requirement was the **"Hybrid (recommended for an
indie)"** option: hex-flavoured directory layout with interfaces only at
the boundaries that benefit (repositories, message bus, time clock).
Domain stays pure Go.

## Decision

Reorganise the backend into the following layout:

```
cmd/supremacy/                  binary entry, mode dispatch
internal/
  app/                          composition root, one file per binary mode
  platform/                     cross-cutting infra (log, metrics, health, httpserver)
  domain/                       pure Go, no infra
  service/                      use-case orchestrators
  adapter/                      drivers (httpapi) + driven (pgrepo, redisrepo, natsbridge)
  engine/  gateway/  worker/    binary-mode composition roots
  auth/  config/  storage/      cross-layer plumbing
pkg/                            importable from outside (api/, errs/, shared/)
```

with the dependency rule:

- `domain` may import only `pkg/`.
- `service` may import `domain`, `pkg`, and `platform` for logging /
  metrics. It declares interfaces that adapters fulfil; it does not
  import driven adapters.
- driven adapters (`pgrepo`, `redisrepo`, `natsbridge`) may import
  `domain`, `platform`, `pkg`. They may not import `service`.
- driving adapters (`httpapi/v1`) may import `service` (they exist to
  call into it). They are listed as a `depguard` exception.
- `app/` may import everything below it.
- `pkg/` may not import `internal/`.

Patterns applied to remove the smells:

- **Strategy** — `unitdom.Kind`, `buildingdom.Kind` replace the
  switch-on-string. Adding a unit = one file + one map entry.
- **Command** — `cmddom.Handler` registry replaces
  `Match.ApplyCommand`'s switch.
- **Repository** — `pgrepo.Users`, `pgrepo.Matches` (formerly
  `match.Repo`, `match.UsersRepo`).
- **Composition root** — every `Run()` is now <80 lines and lives in
  `internal/app/<mode>.go`. The duplicated HTTP boilerplate is in
  `internal/platform/httpserver`.
- **Mapper** — domain `lobbydom.Match` → wire `api.MatchView` mapping
  lives in `internal/adapter/httpapi/v1/mapper.go`. Engine code does not
  change when wire types evolve.

Frontend mirrored the change: feature folders (`features/auth`,
`features/lobby`, `features/match`), `react-router-dom` for shareable
URLs, per-resource API split (`api/auth.ts`, `matches.ts`, `maps.ts`),
`useMatchSocket()` hook owning the WebSocket lifecycle.

Code-quality guardrails locked the new layout in:

- `golangci-lint` v2 with `depguard` enforcing the dependency rules,
  plus `funlen` (80 lines), `gocyclo` (15), `lll` (140 cols), `gosec`,
  `gocritic`, `unparam`, `nilerr`.
- ESLint flat config + Prettier on the web side.
- `lefthook` pre-commit running `gofmt`, `goimports`,
  `golangci-lint run --new-from-rev=main`, and `npm run lint:fix`.
- File / function / package size budgets (see ARCHITECTURE.md).

## Consequences

- **Positive**:
  - Adding a unit type, building, or command is now a one-file change.
  - The dependency direction is enforced; layer leakage breaks the build.
  - HTTP controllers, services and repositories are independently
    testable; the lobby controller does not reach into Postgres.
  - Frontend URLs are now shareable and the back button works.
  - Every package has a `doc.go` so `pkg.go.dev` style overviews work.
- **Negative**:
  - The package count roughly doubled. Onboarding is slower until a new
    contributor reads ARCHITECTURE.md.
  - The `httpapi -> service` exception is a known concession to indie
    pragmatism; a strict hexagonal version would declare driving ports
    in the service package.
  - Typed IDs were introduced gradually (start with `cmddom.Command`).
    Some structs still use bare strings; future ADRs will widen
    coverage.
- **Follow-ups**:
  - Phase 7: revisit JSON vs protobuf for the WebSocket envelope.
  - Add `matchdomtest.NewMatch()...` builder package once the next
    feature lands and we feel the inline `matchdom.New(...)` boilerplate.
  - Audit `internal/auth` for whether it should move into
    `internal/platform/auth`.

## Alternatives considered

- **Strict hexagonal with driving ports.** Cleaner but doubles the
  controller boilerplate. Deferred until we have a second driving
  adapter (e.g. gRPC).
- **Split the binary into multiple modules.** Unnecessary at our size;
  the single-binary mode dispatch is two lines of glue.
- **Introduce a DI framework (wire / fx).** No win at this size; the
  composition root is plain Go and grep-able.
- **Adopt sqlc / openapi-generator.** Out of scope for this refactor;
  evaluate per-feature once the team grows.
