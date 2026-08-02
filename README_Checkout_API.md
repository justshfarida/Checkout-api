# Checkout API

A lightweight RESTful Checkout API built in Go featuring product catalog queries, simulated resilient payment processing, and user order management.

---

## 🚀 Running the Server

```cmd
go run main.go
```

The server will start listening on `http://localhost:8080`.

---

## 🛠 API Endpoints & cURL Examples

> **Note for Windows `cmd.exe` users:** Double quotes (`\"`) inside JSON bodies are escaped using `\\\"`.

---

### 1. Catalog Endpoints

#### A. GET `/items` — List / Filter Catalog Items

```cmd
curl.exe -i -X GET http://localhost:8080/items
```

```cmd
curl.exe -i -X GET "http://localhost:8080/items?category=electronics&sort=price"
```

#### B. GET `/items/{id}`

```cmd
curl.exe -i -X GET http://localhost:8080/items/1
```

### 2. Order Endpoints

#### POST `/orders`

```cmd
curl.exe -i -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d "{\"user_id\":1,\"items\":[{\"item_id\":1,\"quantity\":2}]}"
```

```cmd
curl.exe -i -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d "{\"user_id\":1,\"items\":[{\"item_id\":99999,\"quantity\":1}]}"
```

#### GET `/users/{id}/orders`

```cmd
curl.exe -i -X GET http://localhost:8080/users/1/orders
```
