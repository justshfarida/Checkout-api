Here is your complete, updated **`README.md`** with the new `409 Conflict` idempotency testing scenario included:

```markdown
# Shopping Cart API

A simple RESTful Shopping Cart API built with Go featuring cart TTL aging, item catalog management, and idempotent order processing.

---

## Running the Server

```cmd
go run main.go

```

The server will start listening on `http://localhost:8080`.

---

## API Endpoints & cURL Examples

> **Note for Windows `cmd.exe` users:** Double quotes (`"`) inside JSON bodies are escaped using `\"`.

---

### 0. Setup: Create Initial Cart

Before testing update, delete, and order operations, create a cart for `user_id: 1` containing items `1` (qty: 2) and `2` (qty: 1).

```cmd
curl -i -X POST http://localhost:8080/user/cart -H "Content-Type: application/json" -d "{\"user_id\":1,\"items\":[{\"item_id\":1,\"quantity\":2},{\"item_id\":2,\"quantity\":1}]}"

```

---

## 1. Catalog Endpoints

### GET `/items`

Retrieves the full list of products available in the store catalog.

```cmd
curl -i -X GET http://localhost:8080/items

```

### GET `/items/{id}`

Retrieves detailed information for a specific product by its ID.

```cmd
curl -i -X GET http://localhost:8080/items/1

```

---

## 2. Cart Endpoints

### GET `/user/cart`

Retrieves the current cart for a specific user using the `X-User-ID` header (resets 30-minute inactivity TTL).

```cmd
curl -i -X GET http://localhost:8080/user/cart -H "X-User-ID: 1"

```

### POST `/user/cart`

Creates a new cart or updates items in an existing cart.

```cmd
curl -i -X POST http://localhost:8080/user/cart -H "Content-Type: application/json" -d "{\"user_id\":1,\"items\":[{\"item_id\":1,\"quantity\":2}]}"

```

### PATCH `/user/cart/items/{id}`

* **Successful update:**
```cmd
curl -i -X PATCH http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{\"user_id\":1,\"quantity\":3}"

```


* **Invalid quantity (`400 Bad Request`):**
```cmd
curl -i -X PATCH http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{\"user_id\":1,\"quantity\":0}"

```



### DELETE `/user/cart/items/{id}`

* **Remove one item:**
```cmd
curl -i -X DELETE http://localhost:8080/user/cart/items/2 -H "Content-Type: application/json" -d "{\"user_id\":1}"

```


* **Remove last item (deletes cart):**
```cmd
curl -i -X DELETE http://localhost:8080/user/cart/items/1 -H "Content-Type: application/json" -d "{\"user_id\":1}"

```



---

## 3. Order Endpoints

### POST `/user/orders`

Processes payment and places an order for the user's cart. Requires an `Idempotency-Key` header.

*(Re-run **Step 0** to ensure a fresh cart exists before running order commands!)*

* **Missing `Idempotency-Key` header (`400 Bad Request`):**
```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -d "{\"user_id\":1}"

```


* **Successful initial order (`201 Created`):**
```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -H "Idempotency-Key: order-key-001" -d "{\"user_id\":1}"

```


* **Retry with SAME key and SAME body (`201 Created` — Cached Response):**
```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -H "Idempotency-Key: order-key-001" -d "{\"user_id\":1}"

```


* **Retry with SAME key but DIFFERENT body (`409 Conflict`):**
```cmd
curl -i -X POST http://localhost:8080/user/orders -H "Content-Type: application/json" -H "Idempotency-Key: order-key-001" -d "{\"user_id\":2}"

```



```

```