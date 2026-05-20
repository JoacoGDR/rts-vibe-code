# How to run the project locally

This guide covers the **full stack**: Postgres, Redis, NATS, the five Go services (`core-api`, `gateway`, `engine`, `worker`, `ai-bot`), optional Mailpit, and the **web** SPA.

## Prerequisites

- **Docker** and **Docker Compose** (v2 plugin is fine).
- **Go** 1.25.7 or newer (see `go.mod` and the root `Dockerfile`).
- **Node.js** 20+ and **npm** (for the `web/` app only).

## Option A: Everything in Docker (simplest)

From the repository root:

```bash
docker compose up -d --build
```

Or use the Makefile shortcut:

```bash
make up-all
```

That builds the `supremacy:dev` image, starts Postgres, Redis, NATS, Mailpit, and all five Go commands with the URLs and secrets defined in [`docker-compose.yml`](../docker-compose.yml).

### After the stack is up

| What | URL or address |
|------|----------------|
| Web UI (run separately; see below) | `http://localhost:5173` |
| Core API (REST) | `http://localhost:8080` |
| Gateway (WebSocket) | `ws://localhost:8081/ws` (after ticket; Vite proxies `/ws` for you) |
| Engine health | `http://localhost:8082/healthz` |
| Worker health | `http://localhost:8083/healthz` |
| ai-bot health | `http://localhost:8084/healthz` |
| Postgres | `localhost:5432` (user/password/db: `supremacy` / `supremacy` / `supremacy`) |
| Redis | `localhost:6379` |
| NATS client | `localhost:4222` |
| NATS monitor | `http://localhost:8222` |
| Mailpit (captured email) | `http://localhost:8025` |

Stop and remove containers plus named volumes:

```bash
docker compose down -v
# or
make down
```

## Option B: Hybrid — infra in Docker, Go from your machine

Useful when you want fast rebuilds without rebuilding the image.

1. Start only dependencies:

   ```bash
   make up
   ```

   That runs `docker compose up -d postgres redis nats` (no app containers).

2. In **separate terminals**, from the repo root, run each service (defaults in [`internal/config/config.go`](../internal/config/config.go) already point at `localhost:5432`, `localhost:6379`, `localhost:4222`):

   ```bash
   make run-core-api
   make run-gateway
   make run-engine
   make run-worker
   make run-ai-bot   # optional unless you care about AI slots
   ```

   Or `go run ./cmd/core-api`, etc. The multi-mode binary still works:
   `go run ./cmd/supremacy gateway`.

3. **Metrics ports:** every mode defaults `METRICS_ADDR` to `:9100`. If you run more than one binary on the host, set a unique port per process, for example:

   ```bash
   METRICS_ADDR=:9100 go run ./cmd/core-api
   METRICS_ADDR=:9101 go run ./cmd/gateway
   ```

4. **Mailpit for the worker:** with hybrid setup, Mailpit is not started by `make up`. Either add Mailpit manually (`docker compose up -d mailpit`) and set `SMTP_HOST=localhost` (and `SMTP_PORT=1025`) for the worker, or leave SMTP unset; the worker still persists notifications but skips real SMTP in dev when `SMTP_HOST` is empty (see config comments in `internal/config/config.go`).

5. **JWT and bot key:** defaults match compose (`JWT_SECRET`, `BOT_API_KEY`). Keep them aligned if you mix Docker apps with local apps.

## Web frontend

The SPA does **not** run inside the main compose file. Run it locally:

```bash
make web-install    # once: npm install in web/
make web-dev        # Vite dev server on port 5173
```

[`web/vite.config.ts`](../web/vite.config.ts) proxies browser calls so you do not need CORS tweaks for local dev:

- `/api` → `http://localhost:8080`
- `/ws` → `ws://localhost:8081`

Open **http://localhost:5173** in the browser. Ensure `core-api` and `gateway` are reachable on `8080` and `8081` (Docker full stack or hybrid).

## Useful Makefile targets

| Target | Purpose |
|--------|---------|
| `make up` | Postgres + Redis + NATS only |
| `make up-all` | Full stack via compose |
| `make down` | `docker compose down -v` |
| `make logs` | `docker compose logs -f` |
| `make build` | Build `bin/supremacy` |
| `make test` | Go tests |
| `make web-dev` | Vite dev server |

## Troubleshooting

- **`pull access denied for supremacy`:** The app image is local-only (`supremacy:dev`). Every service that uses it must define the same `build` in `docker-compose.yml` so Compose builds the image instead of pulling from Docker Hub. After a fix, run `docker compose up -d --build` again.
- **Port already in use:** Another Postgres/Redis or an old stack may be bound. Change host ports in `docker-compose.yml` or stop the conflicting service.
- **Gateway cannot reach core-api:** In Docker, services use internal hostnames (`core-api`, `gateway`). From the host, use `localhost` and the published ports above.
- **WebSocket fails from the SPA:** Confirm the gateway is listening on `8081` and you are using the Vite dev URL so `/ws` is proxied.
- **Migrations:** Core-api, engine, and worker run migrations when they connect to Postgres; first boot may take a few seconds until health checks pass.
