# Shopping Cart API

A simple RESTful Shopping Cart API built with Go.

## Running the Server

```cmd
go run main.go
```

The server will start listening on `http://localhost:8080`.

---

## API Endpoints & cURL Examples

### 0. Setup: Create Initial Cart

```cmd
curl -i -X POST http://localhost:8080/user/cart -H "Content-Type: application/json" -d "{"user_id":1,"items":[{"item_id":1,"quantity":2},{"item_id":2,"quantity":1}]}"
```

## 1. Catalog Endpoints

### GET `/items`

```cmd
curl -i -X GET http://localhost:8080/items
```

### GET `/items/{id}`

```cmd
curl -i -X GET http://localhost:8080/items/1
```

## 2. Cart Endpoints

### GET `/user/cart`

```cmd
curl -i -X GET http://localhost:8080/user/cart -H "X-User-ID: 1"
```

### POST `/user/cart`

```cmd
curl -i -X POST http://localhost:8080/user/cart -H "Content-Type: application/json" -d "{"user_id":1,"items":[{"item_id":1,"quantity":2}]}"
```

### PATCH `/user/cart/items/{id}`

Successful:

```cmd
curl -i -X PATCH http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{"user_id":1,"quantity":3}"
```

Invalid quantity:

```cmd
curl -i -X PATCH http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{"user_id":1,"quantity":0}"
```

### DELETE `/user/cart/items/{id}`

Remove one item:

```cmd
curl -i -X DELETE http://localhost:8080/user/cart/items/2 -H "Content-Type: application/json" -d "{"user_id":1}"
```

Remove last item:

```cmd
curl -i -X DELETE http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{"user_id":1}"
```

## 3. Order Endpoints

### POST `/user/orders`

Missing idempotency key:

```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -d "{"user_id":1}"
```

Successful:

```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -H "Idempotency-Key: order-key-001" -d "{"user_id":1}"
```

Retry:

```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -H "Idempotency-Key: order-key-001" -d "{"user_id":1}"
```
