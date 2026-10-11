# Building Resilient Systems — Report

**Author:** Farida Shakikhanli
**Branch:** `FaridaShakikhanli-week11-resilience`
**Environment:** Docker Compose on one Windows laptop (API + Postgres + OTel Collector), k6 on the same machine, Xsolla **sandbox** for payments.

## 1. Summary

- All five patterns are implemented and unit-tested: timeouts, retry with jitter, circuit breaker, graceful shutdown and panic recovery.
- **The service itself is not the bottleneck.** `GET /items` held about **5000 RPS with 0 errors and p99 38.6 ms**.
- **Checkout is limited by Xsolla.** `POST /orders` completed about **11 orders/s**, with a p95 of 8.8 s for the successful ones. Every order waits on a real call to the Xsolla sandbox.
- **Load testing exposed a real bug in my code, now fixed.** A global idempotency lock serialized every checkout (§4.3).
- **Chaos experiment (Postgres frozen, then stopped, at 100 RPS):** the 2 s timeout turned both faults into clean 504s, recovery was automatic, and no error leaked internals. It also showed two gaps: `/health` stayed 200 throughout, and a stopped DB isn't detected faster than a hung one (§5).

## 2. Patterns implemented

| Pattern | Where | What it does |
|---|---|---|
| **Timeouts** | `cmd/api/main.go`, `item_service.go`, `xsolla_service.go` | **Server:** ReadHeader 5 s, Read 10 s, Write 15 s, Idle 60 s. **DB:** startup ping 5 s; item listing query 2 s, returns **504 TIMEOUT**. **Xsolla:** 4 s per HTTP attempt. |
| **Retry + jitter** | `xsolla_service.go` (`createPaymentWithRetry`) | Up to 3 attempts. Wait is `rand(0, 100ms << attempt)` (full jitter), so retries from many requests don't fire at the same moment. Only network errors, 429 and 5xx are retried, never 4xx. Stops waiting when `ctx.Done()` fires. |
| **Circuit breaker** | `internal/resilience/breaker.go`, used in `CreatePayment` | Hand-rolled Closed → Open → Half-Open. **Opens:** after 5 consecutive Xsolla failures, counted after retries. **While open:** fails fast for 30 s with **503 PAYMENT_UNAVAILABLE**. **Half-open:** lets one probe through. A 4xx counts as "Xsolla is healthy". A cancelled request counts as neither success nor failure. |
| **Graceful shutdown** | `cmd/api/main.go` | `signal.NotifyContext` catches SIGINT/SIGTERM. Then `server.Shutdown` gives in-flight requests up to 10 s, telemetry is flushed (10 s), the DB pool is closed, and "Shutdown complete" is logged. |
| **Panic & recover** | `internal/middlewares/recoverMiddleware.go` | `WithRecover` turns a panic into a JSON 500 instead of a dropped connection. It records the panic on the trace span, logs the stack, and re-panics `http.ErrAbortHandler`. |

Tests: `breaker_test.go` (8 tests), `xsolla_retry_test.go`, `xsolla_breaker_test.go`, `item_timeout_test.go`, `recover_test.go`, `error_handler_test.go` and `idempotency_concurrency_test.go`. Run `go test -race ./...`.

## 3. Load test method

- **Tool:** k6, `ramping-arrival-rate` (open model: k6 sends at a fixed rate whether or not the server keeps up).
- **Profile:** 30 s at 50 RPS → 1 min ramp to the target → 1 min hold → 15 s ramp down (2m45s).
- **Scripts:**
  - `loadtest/checkout.js`: `POST /orders`. Logs in, puts one item in the cart, and sends a new `Idempotency-Key` per request.
  - `loadtest/items.js`: `GET /items?limit=20`. Read-only, no Xsolla, shows the service's own capacity.
- **Raw results:** `loadtest/results-*.json`.

## 4. Results

### 4.1 `GET /items`: the service's own capacity

| Target | Requests | Errors | p50 | p95 | p99 | max | Peak VUs | Dropped |
|---|---|---|---|---|---|---|---|---|
| 50 RPS | 7,874 | 0% | 1.96 ms | 4.37 ms | 7.99 ms | 21.7 ms | 1 | 0 |
| 50 → 1000 RPS | 100,499 | 0% | 0.54 ms | 2.33 ms | 5.72 ms | 39.8 ms | 8 | 0 |
| 50 → 5000 RPS | 489,109 | 0% | 1.6 ms | 7.82 ms | 38.6 ms | 120.7 ms | 218 | 1,390 (0.28%) |

- **1000 RPS was easy.** All ~100,500 planned requests were sent, and only 8 were ever in flight at once.
- **At 5000 RPS the first signs of queueing appear.** The median only went from 0.5 to 1.6 ms, but p99 grew **×7** and concurrency **×27**: most requests are still fast, but some now wait for CPU or a DB connection. The 0.28% dropped iterations came from k6 creating new VUs mid-test, not from server errors.
- **The 50 RPS run has a *higher* median than 1000 RPS.** At low rates each request hits a "cold" path: idle CPU cores have to wake up, and the Go runtime and Docker Desktop's network are idle. It doesn't affect users, but it shows that latency isn't simply "lower load = faster".
- **Where the limit is:** above 5000 RPS on this laptop. k6, Docker and Postgres share the same CPU, so above this point the test measures the laptop, not the code.
  <!-- TODO: add `docker stats` CPU of app / db at 5000 RPS -->

### 4.2 `POST /orders`: checkout at 50 RPS

| | Value |
|---|---|
| Requests sent | 7,649 (225 dropped) |
| Successful (201) | **1,861 (24.3%), about 11 orders/s** |
| Failed | 5,788 (75.7%) |
| Successful: p50 / p95 / p99 / max | 3.34 s / 8.81 s / 10.98 s / 12.32 s |
| All requests: p50 / p95 / p99 / max | 4.93 s / 11.18 s / 12.17 s / 12.94 s |
| Peak VUs | 316 |

**Interpretation**

- **The bottleneck is the external dependency.** The same service answers `GET /items` in about 2 ms. Checkout needs a DB write and a real HTTPS call to the Xsolla sandbox, which slows down and fails under load.
- **The maximum of 12.9 s matches the retry budget:** 3 attempts × 4 s timeout + backoff. The slowest requests used every retry before giving up.
- **The fastest failures took 1.5 ms.** That's the circuit breaker answering **503** immediately instead of calling Xsolla, which is what it is designed to do.
  <!-- TODO: confirm the 503 share with: docker compose logs app | findstr "Circuit breaker state changed" -->
- **Raising the rate wouldn't measure anything useful.** Checkout throughput is set by Xsolla, so a 1000 RPS checkout test would only produce more 503s.

### 4.3 Bug found by load testing: global idempotency lock (fixed)

- **First run:** only **29 of 3,798** orders succeeded, and the rest timed out after 60 s.
- **Cause:** `CreateOrder` held one global `sync.Mutex` for the whole checkout, including the Xsolla call. Every request, for every user, waited in a single queue.
- **Fix (commit `d900c173`):** take the lock only to read or write the cache. Store an "in progress" marker for the key, release the lock, call the service, then store the result. A duplicate key sent while the first request is still running gets **409 REQUEST_IN_PROGRESS**.
- **Tests:** `TestCreateOrder_DifferentKeysRunInParallel` and `TestCreateOrder_SameKeyInFlightReturns409`, both run with `-race`.
- **Result after the fix:** about 11 orders/s, now limited by Xsolla instead of my lock.

## 5. Chaos experiment: database failure under load

**Setup:** `.\loadtest\chaos.ps1` runs `loadtest/chaos.js`: `GET /items` at a constant 100 RPS for 5 minutes, plus `GET /health` at 2 RPS to see what a Kubernetes probe would see. A background job breaks Postgres on a fixed schedule:

| Time | Phase | Fault |
|---|---|---|
| 0–60 s | steady | none (steady state) |
| 60–105 s | db_paused | `docker compose pause db`: Postgres is frozen and accepts connections but never answers (like a hung DB or network partition) |
| 105–165 s | recovery_1 | `docker compose unpause db` |
| 165–210 s | db_stopped | `docker compose stop db`: Postgres is gone and connections are refused (like a crash) |
| 210–300 s | recovery_2 | `docker compose start db` |

### Hypothesis vs. observed

| Question | Expected | Observed |
|---|---|---|
| Does the timeout fire? | **Yes, when paused:** about 2 s, **504 TIMEOUT**. | ✅ **Yes.** 4,122 × 504 at p50 2.11 s / max 2.20 s. Nothing hung longer. |
| DB down? | Fast **500 INTERNAL_ERROR**. | ❌ **Wrong.** 4,366 × **504** at about 2.1 s, and only 20 × 500. A stopped DB looked exactly like a hung DB to the app (see below). |
| Does the breaker trip? | **No** (it only wraps Xsolla). | ✅ **No.** Zero 503s; nothing protects the DB call except the timeout. |
| Is the error clean? | JSON without internals. | ✅ **8,523 / 8,523** error bodies were clean JSON with an `error` code and no SQL, host or driver text. |
| Recovery? | Automatic, within seconds. | ✅ **Automatic, no app restart.** After unpause: 6,000 / 6,000 OK. After start: 15 × 504 in the first ~2 s while Postgres booted, then all OK. |
| `/health`? | 200 the whole time. | ✅ **601 / 601 were 200**, including during both outages. |

### Observed

Fault log (seconds since test start): pause 60.2 s → unpause 105.2 s → stop 165.5 s → start 210.4 s.

| Phase | GET /items | 200 | 500 | 504 | p50 | p95 | max | /health 200 |
|---|---|---|---|---|---|---|---|---|
| steady | 6,000 | 6,000 | 0 | 0 | 1.6 ms | 3.5 ms | 20 ms | 120/120 |
| db_paused | 4,331 | 209 | 0 | 4,122 | 2.11 s | 2.14 s | 2.20 s | 90/90 |
| recovery_1 | 6,000 | 6,000 | 0 | 0 | 1.6 ms | 4.5 ms | 1.15 s | 120/120 |
| db_stopped | 4,500 | 114 | 20 | 4,366 | 2.10 s | 2.13 s | 2.15 s | 90/90 |
| recovery_2 | 9,001 | 8,986 | 0 | 15 | 1.4 ms | 3.9 ms | 2.11 s | 181/181 |

How to read it:

- **The 2 s timeout was the only thing that protected the service, in both faults.** Without it, requests would have waited for the 15 s write timeout or the OS TCP timeout.
- **The 200s in the outage phases are not real successes during the outage.** They are the ~2 s of requests that started just before the fault. The phase is assigned when a request *starts*. The same applies to the 504s at the start of recovery_2.
- **Why "DB down" gave 504 instead of 500:** once the `db` container is stopped, connecting to it doesn't get an immediate "connection refused". The DNS lookup or TCP connect inside Docker hangs instead. The pgx pool has no `connect_timeout`, so the attempt waits until the 2 s request deadline fires. Only the 20 requests that hit already-open connections at the moment of the stop failed fast (500). **Lesson:** a dead dependency often looks like a slow one, so timeouts matter more than error handling.
- **Every failing request held a goroutine for 2 s.** The paused phase sent 4,331 instead of 4,500 requests: k6 had to grow from 50 to ~220 VUs (100 RPS × 2.1 s ≈ 210 requests in flight) and dropped 169 iterations while doing so. At 100 RPS that's fine. At 5,000 RPS it would be about 10,000 goroutines waiting on a dead DB.
- **Recovery is instant and needs no restart.** p95 is back at single-digit ms immediately. The 1.15 s max right after unpause is the requests that were frozen being released at once.
- **`/health` hides the outage.** It returned 200 for every check while 95–97% of real requests failed. A Kubernetes readiness probe on `/health` would keep sending traffic to this pod.

### Proposed fixes

1. **Add a readiness endpoint.** `GET /ready` should `pool.Ping` with a ~500 ms timeout and return 503 on failure. Point the Kubernetes `readinessProbe` at it and keep `/health` as the `livenessProbe`. During a DB outage, pods leave the load balancer instead of answering 504s, and they aren't restarted for a problem a restart can't fix.
2. **Fail fast instead of waiting 2 s.** Set `connect_timeout=1` in the DB URL (or `pgxpool.Config.ConnConfig.ConnectTimeout`), so a dead DB is detected faster than the query deadline. Add a circuit breaker around DB calls (the same `resilience.Breaker`) so that after N timeouts requests get **503** immediately for a cooldown, instead of holding a goroutine for 2 s each.
3. **Return 503, not 504/500, when the DB is unreachable.** Map connection errors and an open DB breaker to `503 SERVICE_UNAVAILABLE` with `Retry-After`. 504 suggests "slow", 500 suggests "bug", but this is "temporarily unavailable, retry later".

## 6. Other findings and known limitations

- **Orphan pending orders.** The order row is written *before* the Xsolla call. When the payment call fails or the breaker is open, the order stays `pending` forever. **Fix:** check `breaker.Allow()` before writing the order, and mark the order `failed` when payment creation fails.
- **Idempotency cache is in memory, per pod.** With 2 replicas, a retried request can reach the other pod and create a second order. This is the same problem as the signup kill switch (option B). **Fix:** move the source of truth to Postgres (a unique key on `(user_id, idempotency_key)`) or Redis.
- **Test environment:** one laptop, so there is no real network latency, and k6 competes with the API for CPU. Numbers show relative behaviour, not production capacity.

## 7. How to reproduce

```powershell
docker compose up -d --build
k6 run -e MAX_RPS=1000 --summary-export=loadtest/results-items.json loadtest/items.js
k6 run -e EMAIL=... -e PASSWORD=... -e MAX_RPS=50 --summary-export=loadtest/results-50.json loadtest/checkout.js
powershell -ExecutionPolicy Bypass -File .\loadtest\chaos.ps1
```
