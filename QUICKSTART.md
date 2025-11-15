# Quick Start Guide

## 🚀 Get Started in 5 Minutes

### Step 1: Start Services

```bash
cd d:\projek\order-service
docker-compose up -d
```

Wait for all services to be healthy (~30-60 seconds).

### Step 2: Initialize Database

```bash
# Run migrations
docker exec -i order-postgres psql -U app -d app < migrations/001_init_schema.sql

# Seed sample data
docker exec -i order-postgres psql -U app -d app < migrations/002_seed_data.sql
```

Or using Make:
```bash
make migrate
make seed
```

### Step 3: Verify Services

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","time":1700000000}
```

### Step 4: Create Your First Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 100,
    "items": [
      {
        "product_id": 1,
        "quantity": 2
      }
    ],
    "payment_method": "mock"
  }'
```

Expected response:
```json
{
  "order_id": 1,
  "status": "RESERVED"
}
```

### Step 5: Check Order Status

Wait a few seconds for payment processing, then:

```bash
curl http://localhost:8080/api/v1/orders/1
```

You should see the order status as `CONFIRMED` or `PAID`.

## 🎯 What's Available

### Sample Products (from seed data)

| ID | SKU | Name | Price (cents) | Stock |
|----|-----|------|---------------|-------|
| 1 | LAPTOP-001 | Gaming Laptop RTX 4070 | 1500000 | 100 |
| 2 | PHONE-001 | Smartphone Pro Max | 1200000 | 100 |
| 3 | HEADSET-001 | Wireless Gaming Headset | 50000 | 100 |
| 4 | MOUSE-001 | RGB Gaming Mouse | 30000 | 100 |
| 5 | KEYBOARD-001 | Mechanical Keyboard | 80000 | 100 |

### Monitoring Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **Metrics Endpoint**: http://localhost:8080/metrics

## 🧪 Test Scenarios

### 1. Test Idempotency

Send the same request twice:

```bash
# First request
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-123" \
  -d '{
    "user_id": 200,
    "items": [{"product_id": 1, "quantity": 1}],
    "payment_method": "mock"
  }'

# Second request (should return same order)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-123" \
  -d '{
    "user_id": 200,
    "items": [{"product_id": 1, "quantity": 1}],
    "payment_method": "mock"
  }'
```

Both should return the same `order_id`.

### 2. Test Concurrent Orders

Run load test:

```bash
# Install k6 first: https://k6.io/docs/getting-started/installation/

k6 run tests/load/order_test.js
```

### 3. Test Oversell Protection

```bash
k6 run tests/load/oversell_test.js
```

This will create 500 concurrent orders for the same product and verify no oversell occurs.

### 4. Monitor Order Lifecycle

Watch logs in real-time:

```bash
docker-compose logs -f order-service
```

You'll see:
1. Order created
2. Inventory reserved
3. Payment processing
4. Order confirmed (or cancelled if payment fails)

## 📊 Check Metrics

View key metrics:

```bash
# Total orders created
curl -s http://localhost:8080/metrics | grep orders_created_total

# Total orders paid
curl -s http://localhost:8080/metrics | grep orders_paid_total

# Inventory reservation latency
curl -s http://localhost:8080/metrics | grep inventory_reserve_latency
```

## 🔍 Distributed Tracing

1. Create some orders
2. Open Jaeger UI: http://localhost:16686
3. Select service: `order-service`
4. Click "Find Traces"
5. Explore the complete order flow with timing details

## 🛑 Stop Services

```bash
docker-compose down

# To remove volumes (warning: deletes all data)
docker-compose down -v
```

## 🔄 Reset Everything

```bash
make db-reset
```

This will:
1. Stop all services
2. Delete database volumes
3. Restart services
4. Run migrations
5. Seed sample data

## 📝 Next Steps

- Read [API Documentation](docs/API.md)
- Check [Deployment Guide](docs/DEPLOYMENT.md)
- Explore the code in `internal/` directory
- Add custom products and test scenarios
- Configure Grafana dashboards
- Run performance benchmarks

## ❓ Common Issues

**Port already in use**:
```bash
# Check what's using port 8080
netstat -ano | findstr :8080

# Kill the process or change PORT in .env
```

**Kafka not ready**:
```bash
# Wait for Kafka to be healthy
docker-compose logs kafka

# May take up to 60 seconds on first start
```

**Database connection failed**:
```bash
# Verify PostgreSQL is running
docker exec order-postgres psql -U app -d app -c "SELECT 1"
```

## 🎉 Success!

Your high-concurrency order processing system is now running!

Try creating multiple concurrent orders and watch the saga pattern handle the distributed transaction flow with automatic compensation on failures.
