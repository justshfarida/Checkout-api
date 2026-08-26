# Checkout API

Xsolla School training project — a simplified checkout API built with Go.

Students build this service incrementally across lectures, starting from a basic HTTP server and evolving it into a production-ready system with persistence, authentication, observability, and more.

## Prerequisites

- Go 1.21+
- curl or Postman (for testing endpoints)
- A code editor (VS Code, GoLand, etc.)

## Quick Start

```bash
go run main.go
```

The server starts on http://localhost:8080.

## API Endpoints

### GET /items

Returns all available items.

```bash
curl http://localhost:8080/items
```

### POST /orders

Creates an order with mock payment processing.

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "items": [{"item_id": 1, "quantity": 2}]}'
```

## Running Tests

```bash
go test -v ./controllers/
```

## Branch Guide

Each lecture has two branches:

| Branch | Purpose |
|--------|---------|
| `week-XX/lecture-XX` | **Starter** — scaffold with TODOs and pre-written tests. Fork from here at the start of class. |
| `week-XX/lecture-XX-final` | **Final** — completed code matching the lecture. Compare your work against this. |

### Available Branches

- `week-01/lecture-01` — Intro to HTTP and JSON APIs (starter)
- `week-01/lecture-01-final` — Intro to HTTP and JSON APIs (completed)

## Project Structure (Week 1)

```
checkout-api/
├── main.go                  # HTTP server entry point
├── models/models.go         # Domain models (Item, LineItem, Order)
├── store/store.go           # In-memory data storage
├── handlers/handlers.go     # HTTP route handlers
└── handlers/handlers_test.go # Table-driven handler tests
```

## License

Internal — Xsolla School use only.
 
## Week 1 - Curl examples:

### 1) POST cart
```bash
curl -i -X POST http://localhost:8080/user/cart \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "items": [
      { "item_id": 101, "quantity": 2 },
      { "item_id": 102, "quantity": 1 }
    ]
  }'
```


### 2) Update item quantity in a cart (PATCH)
```bash
curl -i -X PATCH http://localhost:8080/user/cart/items/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "quantity": 5
  }'
```
- 400 - Bad Request if quantity <= 0 or the body is malformed.
- 404 -  Not Found if the user has no cart containing that item.
- 200 -  OK with the updated cart on success


### 3) Delete item from a cart (DELETE, 204 when cart becomes empty)
``` bash
curl -i -X DELETE http://localhost:8080/user/cart/items/{id} \
  -H "Content-Type: application/json" \
  -d '{ "user_id": 1 }'
```
- 404 - Not Found — item wasn't in the user's cart.
- 204 - No Content — item removed and it was the last item, so the cart itself was deleted.
- 200 - OK with the remaining cart — item removed but other items remain.


### 4) Create /orders
``` bash
curl -i -X POST http://localhost:8080/orders \
   -H "Content-Type: application/json" \
   -H "Idempotency-Key: 8f14e45f-ceea-467e-a9d6-2d1a3a5e7c1a" \
   -d '{ "user_id": 1 }'
```

- Missing Idempotency-Key → 400
``` bash
curl -i -X POST http://localhost:8080/orders \
    -H "Content-Type: application/json" \
    -d '{ "user_id": 1 }'
```

- Retrying with the same key (same body) → cached response, no double-charge

``` bash
# First request — creates the order
curl -i -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: retry-key-123" \
  -d '{ "user_id": 1 }'

# Second request, identical key + body — returns the SAME cached response
# (same status code and body as the first call; no new order is created,
# no second payment attempt is made)
curl -i -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: retry-key-123" \
  -d '{ "user_id": 1 }'
```

- Same key, different body → 409 Conflict
``` bash
curl -i -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: retry-key-123" \
  -d '{ "user_id": 2 }'
```

### 5) Get orders for a user
- GET /users/{user_id}/orders
``` bash
curl -i http://localhost:8080/users/1/orders
```
