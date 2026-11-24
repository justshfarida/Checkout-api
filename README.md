# Checkout API

Xsolla School training project: a checkout API for a game store, written in Go.
Users sign up, fill a cart and pay through **Xsolla**; payments are confirmed by a
signed Xsolla webhook, and follow-up work (emails, analytics) runs asynchronously
through **RabbitMQ**.

```
Browser ──JWT──▶ checkout-api (:8080) ──SQL──▶ Postgres
Xsolla ──webhook─▶      │   └── outbox relay ──▶ RabbitMQ (order.events, fanout)
                        │                           ├─▶ notification-worker (email)
                        └── HTTPS ──▶ Xsolla API     └─▶ analytics-worker (daily_sales)
```

## Contents

- [Run it](#run-it)
- [Configuration](#configuration)
- [API](#api)
- [Programs in `cmd/`](#programs-in-cmd)
- [Database migrations](#database-migrations)
- [Tests and CI](#tests-and-ci)
- [Course work and reports](#course-work-and-reports)
- [Project structure](#project-structure)

## Run it

Requirements: **Go 1.26.10**, **Docker Desktop** (Compose v2). Optional: k6 (load tests).

```powershell
# 1. Create .env (never committed) – see "Configuration" below
# 2. Start everything
docker compose up -d --build
# 3. Check
curl.exe http://localhost:8080/health
```

| Service | URL / port |
|---|---|
| API | http://localhost:8080 (Swagger UI: http://localhost:8080/swagger/) |
| Postgres | localhost:5432, database `checkout` |
| RabbitMQ UI | http://localhost:15672 (guest / guest, local only) |
| Redis (webhook lab) | localhost:6379 |

Stop with `docker compose down` (data is kept in the `postgres_data` / `rabbitmq_data` volumes).

## Configuration

The API reads its settings from environment variables (`.env` in Docker Compose)
and **refuses to start** if a required one is missing.

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string |
| `JWT_SECRET` | yes | HS256 signing key, **at least 32 bytes** (checked at startup) |
| `FRONTEND_URL` | yes | CORS allowlist and payment return URL |
| `XSOLLA_PROJECT_ID`, `XSOLLA_MERCHANT_ID`, `XSOLLA_API_KEY` | yes | Xsolla Pay Station |
| `XSOLLA_WEBHOOK_SECRET` | yes | Verifies Xsolla webhook signatures |
| `RABBITMQ_URL` | yes | Broker for the outbox relay (set by Compose) |
| `COOKIE_SECURE` | no | Only `false` disables the `Secure` flag on the refresh cookie (local HTTP) |
| `SIMULATE_EMAIL_FAILURE` | no | `true` makes the email worker always fail (DLQ demo) |

Secrets live only in `.env` (git- and docker-ignored), Kubernetes Secrets and GitLab CI variables.
See [SECURITY.md](SECURITY.md).

## API

| Method & path | Auth | Description |
|---|---|---|
| `GET /health` | – | Liveness check |
| `GET /swagger/` | – | API docs |
| `POST /signup` | – | Create account |
| `POST /login` | – | Log in (rate limit 10/min per IP → 429) |
| `POST /token` | cookie | New access token from the refresh token (rotated) |
| `POST /logout` | cookie | Revoke refresh token |
| `GET /items`, `GET /items/cursor`, `GET /items/{item_id}` | – | Catalog (offset / cursor pagination) |
| `GET /user/me`, `GET /user/inventory` | JWT | Current user, purchased items |
| `GET /user/cart`, `GET /user/cart/cursor` | JWT | Cart |
| `PATCH /user/cart/items/{item_id}`, `DELETE …` | JWT | Change / remove cart item |
| `POST /orders` | JWT + `Idempotency-Key` | Create order + Xsolla payment link |
| `GET /user/orders`, `GET /user/orders/cursor`, `GET /user/orders/{order_id}` | JWT | Own orders only (others → 404) |
| `POST /webhook` | Xsolla signature | Payment events (`order_paid`, `order_canceled`, `user_validation`) |

Resilience built in: server timeouts, 2 s DB query timeout (→ 504), Xsolla retry with
jitter + circuit breaker (→ 503), panic recovery (→ 500), graceful shutdown.

## Programs in `cmd/`

| Program | What it does | Run |
|---|---|---|
| `api` | The HTTP API + outbox relay | `docker compose up -d app` |
| `notification_worker` | Consumes `order_notifications`, sends (simulated) emails; idempotent, 3 retries → DLQ | Compose service `notification-worker` |
| `analytics_worker` | Consumes `order_analytics`, updates `daily_sales` exactly once | Compose service `analytics-worker` |
| `fake-webhook` | Sends a signed `order_paid` webhook for a **pending** order | `go run ./cmd/fake-webhook -order <id>` |
| `publish-bad` | Publishes a poison message (DLQ demo) | `go run ./cmd/publish-bad` |
| `dlq-replay` | Lists a DLQ, or moves messages back with `-replay N` (`-1` = all) | `go run ./cmd/dlq-replay [-queue order_analytics] [-replay -1]` |
| `webhook-lab` | Webhooks homework receiver on :8090 (HMAC, JSONL log, dedup, replay, fan-out, DLQ) | `go run ./cmd/webhook-lab` (`DEDUP=redis`, `FAIL_HANDLER=email`) |
| `webhook-sender` | Signed test events with exponential backoff | `go run ./cmd/webhook-sender [-id evt_a -dup] [-bad-signature]` |
| `seed-xsolla` | One-time: uploads catalog items to the Xsolla project | `go run ./cmd/seed-xsolla` |

### Messaging topology

- `order.events` (**fanout**) → `order_notifications`, `order_analytics`
- `dlx` (**direct**) → `order_notifications.dlq`, `order_analytics.dlq`
- Retries and replays go to a single queue via the default exchange.

## Database migrations

Migrations are in `internal/migrations/` (golang-migrate format, `NNNNNN_name.up.sql` / `.down.sql`).
On first start Postgres is seeded from `db/checkout_dump.sql`; then bring the schema up to date:

```powershell
$db = ((Get-Content .env | Select-String '^DATABASE_URL=').ToString() -split '=',2)[1]
docker run --rm -v "${PWD}/internal/migrations:/migrations" --network farida-checkout-api_default `
  migrate/migrate:v4.18.1 -path=/migrations -database $db up
docker compose exec db psql -U postgres -d checkout -c "SELECT * FROM schema_migrations;"
```

## Tests and CI

```powershell
gofmt -l .
go vet ./...
go test ./...            # CI runs: go test -race ./...
```

| Kind | How |
|---|---|
| Unit tests | `go test ./...` (handlers, services, breaker, workers, outbox, webhook lab) |
| Load tests (k6) | `k6 run -e MAX_RPS=1000 loadtest/items.js`, `loadtest/checkout.js` |
| DB chaos experiment | `powershell -ExecutionPolicy Bypass -File loadtest\chaos.ps1` |
| Queue chaos test | `powershell -ExecutionPolicy Bypass -File scripts\chaos-test.ps1` (kills RabbitMQ + workers, checks no order is lost) |

GitLab CI (`.gitlab-ci.yml`): gofmt → govulncheck + gitleaks (blocking) + gosec →
`go test -race` with ≥ 60 % coverage on `internal/services` → build → Docker image → trivy (blocks on CRITICAL).

## Course work and reports

| Topic | Where |
|---|---|
| Security: STRIDE threat model, scans, hardening, secrets | [SECURITY.md](SECURITY.md) |
| Resilience: patterns, load test, chaos experiment | [docs/resilience-report.md](docs/resilience-report.md) |
| Observability | [docs/observability-runbook.md](docs/observability-runbook.md) |
| Queues: idempotent consumer, outbox, DLQ, fanout, chaos test | `internal/messaging`, `internal/outbox`, `cmd/*_worker`, `scripts/chaos-test.ps1` |
| Webhooks: HMAC, dedup (sync.Map / Redis), replay, fan-out, DLQ | `cmd/webhook-lab`, `cmd/webhook-sender`, `internal/webhooklab` |

## Project structure

```
checkout-api/
├── cmd/                     # entry points (see "Programs in cmd/")
├── internal/
│   ├── handlers/            # HTTP handlers, error mapping, request decoding
│   ├── middlewares/         # auth, CORS, security headers, rate limit, recover
│   ├── services/            # business logic (orders, cart, auth, Xsolla client)
│   ├── repositories/        # interfaces + postgres implementation
│   ├── resilience/          # circuit breaker
│   ├── messaging/           # RabbitMQ topology, publish/consume, dedup
│   ├── outbox/              # transactional outbox relay
│   ├── analytics/           # daily_sales (exactly-once update)
│   ├── notifier/            # simulated email sender
│   ├── webhooklab/          # webhooks homework
│   ├── telemetry/           # OpenTelemetry setup
│   ├── models/, validation/ # shared types, validation helpers
│   └── migrations/          # SQL migrations
├── db/checkout_dump.sql     # initial data for local Postgres
├── docs/                    # Swagger + reports
├── k8s/                     # Kubernetes base + dev/production overlays
├── loadtest/                # k6 scripts, chaos experiment, results
├── scripts/chaos-test.ps1   # queue chaos test
├── Dockerfile               # multi-stage, distroless, nonroot
└── docker-compose.yaml
```

## License

Internal — Xsolla School use only.
