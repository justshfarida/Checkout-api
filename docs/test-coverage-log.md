# Unit test update — coverage log

Run with: `go test -cover ./internal/...`

## Baseline (before)

| Package | Result |
|---|---|
| internal/services | **build failed** (fakes and order tests out of date) |
| internal/handlers/unit_tests | **build failed** (folder is a separate package, so it can't see handler types) |
| internal/handlers | ok, 2.9% |
| internal/middlewares | 0% (no tests) |

## After

| Package | Result |
|---|---|
| internal/services | ok, 93.2% |
| internal/middlewares | ok, 95.6% |
| internal/handlers | ok, all tests pass (all 3 suspected bugs fixed) |

The handler tests moved from `internal/handlers/unit_tests/` to `internal/handlers/`, so they are in the same package as the code they test.

## Iteration log

Format: `candidate — what changed — outcome — reason`

### Repairs to existing tests (needed because the code changed)
- services fakes — added `FindUserByID`, `DeleteRefreshToken`, `GetUserInventory`, `GetOrderItems`, plus Xsolla fakes — kept — tests didn't compile.
- `TestOrderService_CreateOrder` — rewritten for the Xsolla flow. Orders are now created as `pending`, a payment token is requested, and the payment processor is no longer called — kept — the old version expected the removed card-payment flow.
- cart stock tests — fixtures now use `ItemType: "merch"` — kept — the code comment says only merch is limited by stock. **The assertions were not changed.**
- `TestHandler_IssueJWT` — request method changed from GET to POST — kept — the route and the godoc both say `POST /token`.
- `TestHandler_CreateOrder` — the 402 "payment declined" case was removed, and the response is now checked for `payment_url` and `xsolla_order_id` — kept — that's the new contract.

### New tests — services (70.8% → 93.2%)
- `VerifyXsollaSignature` (valid, wrong secret, tampered body, missing scheme, empty) — kept — security-critical; expected values follow Xsolla's documented SHA-1(body + secret).
- `CreatePayment` request shape (URL, basic auth, user id, items, `custom_parameters.order_id`) plus the non-2xx error — kept — the webhook depends on `order_id` coming back.
- `ProcessXsollaPayment` (items forwarded, empty order, repo errors, duplicate webhook not treated as an error) — kept.
- `UpdateOrderStatus`, `GetOrdersByUserIDOffset/Cursor`, `GetUserCartCursor` (limit+1 page detection, cursor of last item, invalid cursor) — kept.
- `UserExists` (not found → false with no error, infrastructure error returned) — kept.
- `Logout` deletes the same hash that login saved — kept — otherwise logout wouldn't revoke the token.
- `DecodeCursor` boundaries (empty, bad base64, bad JSON, negative id) — kept.
- "CreateOrder does not clear cart" — **discarded** — rule 10: no reliable evidence for the expected behavior.

### New tests — handlers (72.6% → 89.2%) and middlewares (0% → 95.6%)
- Xsolla webhook: invalid signature rejected before processing, malformed JSON, order_paid / order_canceled / user_validation, unknown type acknowledged — kept.
- Logout, GetCurrentUser (no password hash leaked), GetUserInventory — kept.
- Idempotency: replayed body is identical; a failed request doesn't burn the key — kept.
- AuthMiddleware: valid token, case-insensitive scheme, missing/expired/wrong-secret/alg-none tokens, bad subject — kept.
- CORS: preflight is short-circuited; credentials and needed headers are allowed — kept.

## Suspected production bugs (the failing tests are kept on purpose)

1. ✅ **FIXED** — **Idempotency key is not scoped to the user**: `order_handler.go`, `TestHandler_CreateOrder_IdempotencyKeyIsScopedToUser`.
   - The cache is keyed only by `Idempotency-Key`. If user 8 sends a key that user 7 already used, user 8 receives user 7's order and payment link.
   - Fix: key the cache by `userID + ":" + key`. The map is also not safe for concurrent requests, so add a `sync.Mutex`.
2. ✅ **FIXED** — **Refresh cookie MaxAge is in nanoseconds**: `auth_handler.go` (LoginUser and IssueJWT), `TestHandler_LoginUser_RefreshCookieLastsSevenDays`.
   - `int(7 * 24 * time.Hour)` = 604800000000000, but `http.Cookie.MaxAge` is in seconds.
   - Fix: `int((7 * 24 * time.Hour).Seconds())`.
3. ✅ **FIXED** — **Webhook user_validation returns 404 on an internal error**: `xsolla_webhook.go`, `TestXsollaWebhook_UserValidation`.
   - The code is labelled `INTERNAL_ERROR` but the status is 404.
   - Fix: return 500, which also lets Xsolla retry.

## Ambiguities (no test added)
- An empty cart on `POST /orders` (`ErrEmptyCart`) currently returns 500. It's probably meant to be a 4xx, but no contract says which code.
- `GET /user/me` returns 500 for a deleted user (404 or 401 might be intended).
- The webhook answers bad signatures with `"INVALID SIGNATURE"` (with a space), in a flat JSON shape. Xsolla's docs show `INVALID_SIGNATURE` nested under `error.code`. Check this against the Xsolla docs.
- If Xsolla fails after the order row is created, the order stays `pending`.

## Intentionally not unit-tested
- `repositories/postgres`: needs a real database, so it belongs in integration tests (testcontainers or the docker-compose Postgres).
- `telemetry`, `cmd/*`: wiring code with no logic worth asserting.
- `MockPaymentProcessor`: no longer used by any code path.
- Pass-through methods `GetUserByID` and `GetUserInventory` in AuthService.

## Other notes
- `docs/docs.go` (Swagger) is stale. It still shows `/auth/refresh`, but the route is now `POST /token`. Run `swag init -g cmd/api/main.go`.
