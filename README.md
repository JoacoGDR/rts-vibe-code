# Supremacy 1914 Clone

A persistent-world RTS inspired by Supremacy 1914. The architecture is a
modular monolith in Go that splits cleanly into separate services as scale
demands. The web client is a React + PixiJS SPA hosted statically.

> Status: **Phase 0 — foundations scaffolded**. See the plan in
> `.cursor/plans/refine-rts-architecture_*.plan.md` for the full roadmap.

## Quick start (local dev)

Prerequisites: Go 1.25.7+, Docker, Node 20+, GNU Make.

```bash
# 1. Start infrastructure (Postgres, Redis, NATS JetStream)
make up

# 2. Run a service locally (in separate shells)
make run-core-api
make run-gateway
make run-engine
make run-worker

# 3. Run the web client
make web-install   # first time
make web-dev       # http://localhost:5173 (proxies /api -> 8080, /ws -> 8081)
```

Health checks: every binary exposes `/healthz` (liveness) and `/readyz`
(readiness) on its HTTP port, plus `/metrics` on a separate Prometheus port
(`9100` for core-api, `9101` gateway, `9102` engine, `9103` worker).

## Layout

```
cmd/
  core-api/ gateway/ engine/ worker/ ai-bot/   # dedicated service binaries
  supremacy/                                   # multi-mode binary (local dev)
internal/
  platform/bootstrap/  # shared process startup
  config/              # env-driven configuration
  observability/       # slog logger, Prometheus metrics, health probes
  coreapi/             # REST API: auth, lobby, match lifecycle
  gateway/             # WebSocket edge, NATS bridge
  engine/              # simulation: timeline, combat, fog of war, economy
  worker/              # background jobs: snapshots, macro-pulse, AI
  storage/             # Postgres / Redis adapters
  auth/                # JWT, sessions, WS tickets
  match/               # match registry & lifecycle
pkg/shared/
  wire/                # JSON wire types between client and gateway
  proto/               # reserved for protobuf if we migrate later
  maps/                # map / scenario YAML
web/                   # React + PixiJS SPA (Vite)
terraform/             # IaC skeleton (Phase 7 turns it on)
migrations/            # goose-style SQL migrations
.github/workflows/     # CI (active) + deploy (scaffolded for Phase 7)
docker-compose.yml     # local dev stack
Dockerfile             # backend image
Makefile
```

## Roadmap

| Phase | Focus                                                     |
| ----- | --------------------------------------------------------- |
|   0   | Foundations: monorepo, dev infra, CI lint+test            |
|   1   | Vertical slice: auth, 1 unit, 1 map, move + combat        |
|   2   | Fog of war (visibility module inside engine)              |
|   3   | Economy: provinces, resources, buildings, multi-unit      |
|   4   | Diplomacy state machine + chat service                    |
|   5   | AI takeover for inactive players + offline notifications  |
|   6   | Match lifecycle, lobby, victory detection                 |
|   7   | Observability, replay log, Terraform/EKS, S3 deploy, k6   |
|   8   | Scale-out: split visibility, sharding, clustered infra    |

Each phase produces a runnable, testable thing.

## Production target (Phase 7)

- Single-region AWS, IaC via Terraform.
- Backend on EKS (Deployments for stateless services, StatefulSet for engine
  because of in-memory match timelines).
- RDS Postgres + ElastiCache Redis (single primary OK for MVP).
- SPA in S3 + CloudFront.
- CI/CD via GitHub Actions with OIDC roles.
- Load-tested to 200 concurrent WS clients across 20 active matches.
