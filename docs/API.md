# Postman Collection for Order Service API

## Setup

Base URL: `http://localhost:8080/api/v1`

## Endpoints

### 1. Health Check
```
GET http://localhost:8080/health
```

### 2. Create Order
```
POST http://localhost:8080/api/v1/orders
Content-Type: application/json

{
  "user_id": 123,
  "items": [
    {
      "product_id": 1,
      "quantity": 2
    },
    {
      "product_id": 2,
      "quantity": 1
    }
  ],
  "payment_method": "mock"
}
```

### 3. Create Order with Idempotency Key
```
POST http://localhost:8080/api/v1/orders
Content-Type: application/json
Idempotency-Key: my-unique-key-12345

{
  "user_id": 456,
  "items": [
    {
      "product_id": 3,
      "quantity": 1
    }
  ],
  "payment_method": "mock"
}
```

### 4. Get Order
```
GET http://localhost:8080/api/v1/orders/1
```

### 5. Get Metrics
```
GET http://localhost:8080/metrics
```

## cURL Examples

### Create Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123,
    "items": [
      {"product_id": 1, "quantity": 2}
    ],
    "payment_method": "mock"
  }'
```

### Get Order
```bash
curl http://localhost:8080/api/v1/orders/1
```

### Health Check
```bash
curl http://localhost:8080/health
```
