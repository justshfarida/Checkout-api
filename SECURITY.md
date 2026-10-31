# SECURITY.md — checkout-api

Owner: Farida Shakikhanli · Last reviewed: 2026-10-12 · Branch: `FaridaShakikhanli-week11-resilience`

## 1. What we are building

A Go checkout API for a game store. Users sign up, log in (JWT access token + refresh-token cookie), fill a cart, and create an order. The API asks **Xsolla** for a payment token, and Xsolla calls back our **`POST /webhook`** when the order is paid or canceled. Data lives in **Postgres**. The service runs in Docker Compose (dev) and Kubernetes (minikube).

```
 Browser ──JWT──▶ ┌────────────────┐ ──SQL──▶ Postgres
                  │  checkout-api  │
 Xsolla ──signed─▶│   (Go, :8080)  │ ──HTTPS──▶ Xsolla API (payment token)
        webhook   └────────────────┘
```

Every arrow in is attacker-controlled until it is validated.

## 2. STRIDE threat model

| | Threat to *this* service | Control we have today | Gap |
|---|---|---|---|
| **S**poofing | Forged or stolen JWT used to act as another user. Fake "order paid" webhook sent by an attacker. | HS256 only (`WithValidMethods`, kills `alg=none`). 15-minute access token. `JWT_SECRET` must be ≥ 32 bytes or the app refuses to start. Refresh token is random, stored **hashed** (SHA-256), rotated on use, in an `HttpOnly; Secure; SameSite=Lax` cookie. Webhook signature `SHA-1(body + secret)` checked in constant time **before** the body is parsed. | No secret rotation process (no key ID / overlap window). |
| **T**ampering | Client changes the price or total of an order. SQL injection through query/body. Webhook body changed in transit. | Total is computed server-side from DB prices; the client never sends a price. All SQL uses `$1` parameters (pgx). Unknown JSON fields rejected (`DisallowUnknownFields`). Webhook signature covers the exact raw bytes. | — |
| **R**epudiation | "I never placed that order." | Every order row has `user_id`; idempotency keys stop double orders; structured logs with `trace_id` go to Datadog. | **No dedicated audit log** for login success/failure and token refresh (who, what, when, IP). |
| **I**nformation disclosure | Reading another user's order by changing the ID. Error messages leaking SQL or hosts. Secrets leaking via git or images. | `GET /user/orders/{id}` filters with `WHERE id = $1 AND user_id = $2`; "not yours" = "not found" = 404 (tested). Unknown errors return a generic 500, details only in logs (tested). Image is distroless with `.dockerignore`; gitleaks in CI. | **App connects to Postgres as the `postgres` superuser** with the default password. An injection bug would expose every table and allow `DROP`. |
| **D**enial of service | Huge request bodies; slowloris clients; slow DB or Xsolla tying up goroutines; password brute force. | 1 MB body cap → 413. Server timeouts (ReadHeader 5 s, Read 10 s, Write 15 s, Idle 60 s). DB query 2 s → 504. Xsolla 4 s per attempt, 3 attempts, circuit breaker → 503. `POST /login` limited to 10/min per IP → 429. | **Rate limit is per pod and keyed by TCP peer IP.** Behind the ingress every user has the ingress IP, so one attacker can lock *everyone* out of login; with 2 replicas the real limit is 20/min. |
| **E**levation of privilege | Calling routes you shouldn't; adding `"is_admin": true` to a body; code running as root in the container. | Every `/user/*` and `/orders` route requires a valid JWT (deny by default). Unknown fields → 400. No admin routes exist. Container runs as `nonroot` on distroless (no shell). | **Debug routes** `/telemetry-test-error`, `/telemetry-test-panic` and `/swagger/` are reachable in every environment. |

### Top 3 gaps → tasks

1. **Least-privilege database user.** Create a `checkout_app` role with only `SELECT/INSERT/UPDATE` on its own tables (no `DROP`, no other schemas) and a generated password; the API uses it, migrations keep the owner role.
2. **Rate limiter that works behind the ingress.** Read the client IP from `X-Forwarded-For` **only** when the request comes from our own ingress, and move the counters to Redis so all replicas share one limit. Until then the limit protects Docker Compose but is a lock-out risk in k8s.
3. **Debug endpoints off outside dev.** Register `/telemetry-test-*` and `/swagger/` only when `APP_ENV=dev`.

(Next after these: an audit log for login and token events.)

## 3. Scans (task 02)

Run locally with:

```powershell
govulncheck ./...
gosec -severity medium ./...
docker run --rm -v "${PWD}:/repo" zricethezav/gitleaks:latest detect --source /repo --log-opts="HEAD" --redact -v
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:latest image --severity HIGH,CRITICAL farida-checkout-api-app
```

| Tool | Before | After | What was done |
|---|---|---|---|
| govulncheck | **38** reachable vulnerabilities (Go stdlib 1.26.0) + `x/net` v0.59.0 | **0** | Go 1.26.9 (`go.mod`, Dockerfile, CI), `x/net` v0.60.0 |
| govulncheck (2026-10-30) | **1** new stdlib vulnerability GO-2026-6809 (`crypto/x509`, reachable via `amqp091.Dial`), caught by the blocking CI job | **0** | Go 1.26.9 → 1.26.10 (`go.mod`, Dockerfile, CI) |
| gosec | 6 issues | **0** (4 `#nosec` with reasons) | Refresh cookie was `Secure: false` → fixed (one helper, Secure by default, `COOKIE_SECURE=false` only for local HTTP) |
| gitleaks | 32 leaks in all 1,098 commits; **1 in this branch** | **0** in this branch | See §5 |
| trivy (new image) | — | **0 HIGH, 0 CRITICAL** (OS: 5 packages; Go binary) | Distroless image, see §5 |

### Accepted findings (suppressed with a written reason)

| Rule | Where | Why it is safe |
|---|---|---|
| G404 weak random | `xsolla_service.go` retry backoff | Random jitter for retry timing, not a secret. Predicting it gains nothing. |
| G401 / G505 SHA-1 | `xsolla_service.go` webhook signature | Xsolla's protocol defines the signature as SHA-1(body + secret); we must match it. Compared in constant time. |
| G124 cookie | `auth_handler.go` `setRefreshCookie` | `Secure` comes from config and is `true` by default (tested); HttpOnly and SameSite are always set. |

The other 31 gitleaks hits are in **other students' branches** (test fixtures, README curl examples, their docker-compose dev secrets). They are not in this branch's history; CI scans only `HEAD`.

### CI gate (`.gitlab-ci.yml`)

| Job | Stage | Blocking? |
|---|---|---|
| govulncheck | security | yes |
| gitleaks (this branch's full history) | security | yes |
| gosec | security | `allow_failure` until **2026-11-01**, then blocking |
| trivy on the pushed image | scan | yes on CRITICAL, HIGH reported |

## 4. HTTP hardening (task 03) and access control (task 04)

| Control | Proof (test) |
|---|---|
| `http.Server` timeouts set (5 / 10 / 15 / 60 s) | `cmd/api/main.go` |
| 1 MB body limit + `DisallowUnknownFields` on every JSON endpoint and the webhook | `TestBodyOverOneMegabyteIsRejectedWith413` (10 MB → 413), `TestLogin_UnknownFieldIsRejected` |
| Security headers: HSTS, nosniff, `X-Frame-Options: DENY`, CSP `default-src 'none'`, `no-referrer`, `no-store` | `TestSecurityHeaders`, `TestSecurityHeaders_SwaggerKeepsWorking` |
| CORS from `FRONTEND_URL` allowlist with credentials (never `*`) | `TestCORS` |
| Login rate limit 10/min per IP, `Retry-After` | `TestLogin_EleventhAttemptInAMinuteGets429` (11th → 429) |
| Ownership in the query for `GET /user/orders/{id}` | `TestGetUserOrder_AccessControl`: owner → 200, other user → 404, anonymous → 401; `..._MissingAndForeignOrderLookIdentical` |
| Idempotency key on `POST /orders`, scoped per user | existing handler tests |

## 5. Secrets and container (task 05)

### Incidents found and handled

| When found | What | Impact | Action |
|---|---|---|---|
| 2026-10-12, gitleaks | k6 `--summary-export` wrote the load-test user's **access token** into `loadtest/results-50.json` (commit `179d3995`). | 15-minute JWT, already expired. Does not reveal the signing secret. | Token removed from the file; `checkout.js` now writes its summary without setup data; fingerprint listed in `.gitleaksignore` with this reason. No rotation needed. |
| 2026-10-12, Dockerfile review | Old Dockerfile did `COPY . .` from a single `golang` stage, so **locally built images contained `.env`** (DB URL, JWT secret, Xsolla API key, webhook secret) and all source code. | Anyone with a local image could read every secret. CI images were not affected (`.env` is not in git). | Multi-stage distroless image + `.dockerignore`; Compose now passes secrets at runtime via `env_file`. |
| 2026-10-12, new startup check | `JWT_SECRET` was 31 bytes (below the 32-byte minimum for HS256), and other students had committed JWT secrets in their branches. | A short secret can be brute-forced offline from one token. | **Rotated** to a random 64-character value; app now refuses to start with < 32 bytes. |

### Where secrets live

* **Local:** `.env` (git-ignored, docker-ignored), passed to the container with `env_file`.
* **Kubernetes:** a `Secret` (`k8s/base/secret.yaml` is git-ignored).
* **CI:** GitLab masked/protected variables (`CI_REGISTRY_*`).
* **Never** in the image, the repo, or logs. Tokens and passwords are not logged; refresh tokens are stored only as hashes.

### Container

* Build stage `golang:1.26.10`, runtime `gcr.io/distroless/static-debian12:nonroot`.
* Static binary (`CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"`), runs as `nonroot:nonroot`, no shell, no package manager.
* `.dockerignore` excludes `.git`, `.env*`, keys, the k8s secret file and build leftovers.
* trivy: **0 HIGH / 0 CRITICAL**.

### Config fails fast

The app exits at startup if any of `DATABASE_URL`, `JWT_SECRET`, `FRONTEND_URL`, `XSOLLA_*` is missing, or if `JWT_SECRET` is shorter than 32 bytes (`TestValidateJWTSecret`).

## 6. Already in place (bonus items)

* Passwords hashed with **bcrypt**.
* JWT parsed with `WithValidMethods(HS256)`, access token **15 minutes**, refresh token 7 days, rotated and revocable.
* No SSRF surface: the API only calls one fixed Xsolla host, never a user-supplied URL.

## 7. Reporting a vulnerability

Contact the owner directly (school GitLab); do not open a public issue with exploit details.
