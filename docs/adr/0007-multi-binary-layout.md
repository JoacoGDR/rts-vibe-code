# ADR-0007: Multi-Binary Layout

- **Status**: accepted
- **Date**: 2026-05-19
- **Deciders**: @joaquing
- **Phase**: code-quality / operational (multi-binary decomposition)

## Context

The backend already ran as five logical services in Docker Compose, all
from one `supremacy` artifact selected by a mode flag
(`core-api`, `gateway`, `engine`, `worker`, `ai-bot`). That kept deploys
simple but made the entry graph harder to navigate for contributors and
linked every service to the full dependency graph at build time.

ADR-0001 deferred splitting binaries until the layered layout was stable;
that layout is now in place.

## Decision

- Add **dedicated entrypoints** under `cmd/<service>/main.go`, one per
  production service, each calling
  [`internal/platform/bootstrap`](../internal/platform/bootstrap/bootstrap.go)
  with a fixed `config.Mode*` and the existing composition-root `Run`
  function.
- Keep **`cmd/supremacy`** as a multi-mode binary for local dev and
  integration tests (`supremacy gateway`, `SUPREMACY_MODE`, etc.).
- Build all artifacts from the **same `go.mod`** (no `go.work`); shared
  code stays in `internal/` and `pkg/` per ADR-0001.
- Docker image ships every binary under `/usr/local/bin/`; Compose
  services invoke the matching binary directly.
- **Postgres migrations** run only in **core-api** (`storage.Migrate`);
  worker and ai-bot replicas must not race goose.

## Consequences

- **Positive**:
  - Clear “start here” per service for onboarding.
  - `make build-all` / CI can verify each binary compiles independently.
  - Production commands no longer depend on a mode flag typo.
  - Worker/ai-bot scaling no longer risks duplicate migration runs.
- **Negative**:
  - Docker image still contains all binaries until per-service images
    are adopted (optional follow-up).
  - Five `main.Version` ldflags targets if build scripts are customized
    per service (today one `LDFLAGS` pattern is shared).
- **Follow-ups**:
  - Per-service Docker images for smaller deploy units.
  - Trim unused deps per binary with build tags only if image size matters.

## Alternatives considered

- **`go.work` + multiple modules** — rejected; unnecessary for a single
  team and one release train.
- **Remove `cmd/supremacy`** — rejected; useful for `go run ./cmd/supremacy
  engine` without five terminal profiles.
- **Migrate in a one-shot init container** — deferred; core-api as owner
  matches current Compose startup order.
