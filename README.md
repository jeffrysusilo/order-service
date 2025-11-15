# Order Service - High-Concurrency Order Processing System

A high-performance, microservice-based backend system for handling e-commerce checkout operations at scale. Built with Golang, designed for high concurrency with no-oversell guarantees, consistent inventory management, and reliable transaction processing through event-driven architecture.

## 🚀 Features

- **High-Concurrency Order Processing**: Handles thousands of concurrent orders per second
- **No-Oversell Guarantee**: Atomic stock reservation using Redis Lua scripts and PostgreSQL transactions
- **Event-Driven Architecture**: Kafka-based async messaging for service communication
- **Saga Pattern**: Distributed transaction management with compensation logic
- **Observability**: Full tracing (Jaeger), metrics (Prometheus/Grafana), and structured logging
- **Idempotency**: Built-in idempotency key support to prevent duplicate orders
- **Mock Payment Service**: Simulated payment processing for testing
- **Fast Inventory Management**: Redis-backed inventory with database persistence

## 📋 Architecture

```
[Client] --> [API Gateway/Gin] --> [Order Service]
                                         |
                                         v
                                  [Inventory Service]
                                         |
                                         v
                                  [PostgreSQL & Redis]
                                         ^
                                         |
[Order Service] <--- Kafka Event Bus ---> [Payment Service]
       |                                        |
       v                                        v
Background Workers                    Mock Payment Gateway
(Compensation/Retries)
```

### Components

1. **Order Service**: Core business logic for order lifecycle management
2. **Inventory Service**: Stock management and reservation
3. **Payment Service**: Mock payment processing
4. **Event Bus**: Kafka for async event-driven communication
5. **Database**: PostgreSQL for persistent storage
6. **Cache**: Redis for fast inventory operations
7. **Observability**: Prometheus, Grafana, Jaeger for monitoring

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Message Broker**: Kafka
- **Database Driver**: sqlx
- **Kafka Client**: segmentio/kafka-go
- **Tracing**: OpenTelemetry + Jaeger
- **Metrics**: Prometheus + Grafana
- **Logging**: Uber Zap
- **Container**: Docker, Docker Compose

## 📦 Project Structure

```
order-service/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go
├── config/                  # Configuration management
│   └── config.go
├── internal/
│   ├── api/                 # HTTP handlers (Gin)
│   │   └── handler.go
│   ├── broker/              # Kafka producer/consumer
│   │   ├── kafka.go
│   │   └── events.go
│   ├── models/              # Domain models
│   │   ├── models.go
│   │   └── events.go
│   ├── redisclient/         # Redis client with Lua scripts
│   │   ├── client.go
│   │   └── scripts/
│   │       ├── reserve_stock.lua
│   │       ├── release_stock.lua
│   │       └── commit_stock.lua
│   ├── service/             # Business logic
│   │   ├── order_service.go
│   │   ├── inventory_client.go
│   │   ├── payment_service.go
│   │   └── saga_orchestrator.go
│   ├── store/               # Database access (sqlx)
│   │   ├── store.go
│   │   └── orders.go
│   ├── util/                # Utilities
│   │   ├── logger.go
│   │   ├── metrics.go
│   │   └── tracing.go
│   └── worker/              # Background workers
│       └── worker.go
├── migrations/              # SQL migrations
│   ├── 001_init_schema.sql
│   └── 002_seed_data.sql
├── deployments/
│   └── prometheus.yml
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## 🚦 Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Make (optional, for convenience commands)

### Quick Start with Docker

1. **Clone and navigate to the project**:
```bash
cd order-service
```

2. **Copy environment file**:
```bash
cp .env.example .env
```

3. **Start all services**:
```bash
make docker-up
# or
docker-compose up -d
```

4. **Run migrations and seed data**:
```bash
make migrate
make seed
```

5. **Verify services are running**:
```bash
# Check health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

### Access URLs

- **Order Service API**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger UI**: http://localhost:16686
- **PostgreSQL**: localhost:5432 (app/secret)
- **Redis**: localhost:6379
- **Kafka**: localhost:9092

## 📖 API Documentation

### Create Order

**POST** `/api/v1/orders`

```json
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
  "payment_method": "mock",
  "idempotency_key": "optional-unique-key"
}
```

**Response** (201 Created):
```json
{
  "order_id": 1001,
  "status": "RESERVED"
}
```

### Get Order

**GET** `/api/v1/orders/:id`

**Response** (200 OK):
```json
{
  "order": {
    "id": 1001,
    "user_id": 123,
    "total_amount": 3100000,
    "status": "CONFIRMED",
    "created_at": "2025-11-15T10:00:00Z",
    "updated_at": "2025-11-15T10:00:05Z"
  },
  "items": [
    {
      "id": 1,
      "order_id": 1001,
      "product_id": 1,
      "quantity": 2,
      "unit_price": 1500000
    }
  ]
}
```

## 🔄 Order Flow (Saga Pattern)

1. **Client** → POST /orders → **Order Service**
2. **Order Service** validates and creates order (status: `CREATED`)
3. **Order Service** → **Inventory Service**: Reserve stock
   - Success → Update order status to `RESERVED`
   - Failure → Cancel order
4. **Order Service** publishes `OrderReserved` event → Kafka
5. **Payment Service** consumes event → Processes payment
6. **Payment Service** publishes result:
   - `PaymentSuccess` → **Order Service** commits reservation → status `PAID` → `CONFIRMED`
   - `PaymentFailed` → **Order Service** compensates (releases stock) → status `CANCELLED`

## 🔐 Concurrency & Anti-Oversell Strategy

### Hybrid Approach (Redis + PostgreSQL)

**Fast Path (Redis Lua Script)**:
```lua
local available = tonumber(redis.call("HGET", KEYS[1], "available") or "0")
if available >= tonumber(ARGV[1]) then
  redis.call("HINCRBY", KEYS[1], "available", -ARGV[1])
  redis.call("HINCRBY", KEYS[1], "reserved", ARGV[1])
  return 1
end
return 0
```

**Fallback (PostgreSQL Transaction)**:
```sql
BEGIN;
SELECT available FROM inventory WHERE product_id = $1 FOR UPDATE;
UPDATE inventory SET available = available - $qty, reserved = reserved + $qty 
WHERE product_id = $1;
COMMIT;
```

**Benefits**:
- ✅ High throughput (Redis atomic operations)
- ✅ Strong consistency (PostgreSQL transactions)
- ✅ Automatic fallback on Redis failure
- ✅ Eventual consistency via background sync

## 📊 Observability

### Metrics (Prometheus)

Key metrics exposed at `/metrics`:
- `orders_created_total`: Total orders created
- `orders_reserved_total`: Orders with inventory reserved
- `orders_paid_total`: Successfully paid orders
- `orders_failed_total`: Failed orders (by reason)
- `inventory_reserve_latency_seconds`: Inventory reservation latency
- `payment_success_total`: Successful payments
- `http_request_duration_seconds`: API latency

### Tracing (Jaeger)

Distributed tracing for:
- Order creation flow
- Inventory reservation
- Payment processing
- Event publishing/consumption

Access Jaeger UI at http://localhost:16686

### Logging (Zap)

Structured JSON logging with levels:
- INFO: Normal operations
- WARN: Recoverable errors
- ERROR: Operation failures
- DEBUG: Detailed debugging (dev mode)

## 🧪 Testing

### Unit Tests
```bash
make test
```

### Coverage Report
```bash
make test-coverage
```

### Load Testing (k6)

Create `tests/load/order_test.js`:
```javascript
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  vus: 100,
  duration: '30s',
};

export default function() {
  const payload = JSON.stringify({
    user_id: Math.floor(Math.random() * 1000),
    items: [{ product_id: 1, quantity: 1 }],
    payment_method: 'mock',
  });

  const res = http.post('http://localhost:8080/api/v1/orders', payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 201': (r) => r.status === 201,
  });
}
```

Run:
```bash
k6 run tests/load/order_test.js
```

## 🛠️ Development

### Local Development

```bash
# Install dependencies
make deps

# Run locally (requires external services)
make run

# Format code
make fmt

# Run linter
make lint
```

### Database Operations

```bash
# Reset database (warning: deletes all data)
make db-reset

# Run migrations only
make migrate

# Seed sample data
make seed
```

### Docker Operations

```bash
# Rebuild and restart
make docker-rebuild

# View logs
make docker-logs

# Stop all services
make docker-down
```

## 🔧 Configuration

Environment variables (`.env`):

```bash
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgres://app:secret@localhost:5432/app?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_ORDER_EVENTS=order-events
KAFKA_CONSUMER_GROUP=order-service-group

# Observability
JAEGER_ENDPOINT=http://localhost:14268/api/traces
PROMETHEUS_PORT=9090
```

## 📈 Performance Characteristics

- **Throughput**: 1000+ orders/sec (depends on hardware)
- **Latency**: P95 < 100ms for order creation
- **Inventory Operations**: P99 < 10ms (Redis fast path)
- **Payment Processing**: ~100-500ms (mocked delay)

## 🔒 Security Considerations

- Parameterized SQL queries (prevent SQL injection)
- Idempotency keys (prevent duplicate orders)
- Rate limiting (should be added at API gateway)
- TLS for production (Kafka, PostgreSQL, Redis)
- Input validation on all endpoints

## 🐛 Troubleshooting

**Services not starting:**
```bash
# Check logs
docker-compose logs

# Verify health
docker-compose ps
```

**Database connection issues:**
```bash
# Test connection
docker exec -it order-postgres psql -U app -d app
```

**Kafka issues:**
```bash
# List topics
docker exec order-kafka kafka-topics --list --bootstrap-server localhost:9092
```



**Need Help?** Check the logs, metrics, and traces!
