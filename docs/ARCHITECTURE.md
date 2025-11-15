# Architecture Documentation

## System Architecture

### High-Level Overview

```
┌─────────────┐
│   Client    │
│ (HTTP/REST) │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│              API Gateway (Gin)                       │
│  - Request validation                                │
│  - Idempotency check                                │
│  - Metrics collection                               │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│            Order Service (Core)                      │
│  - Business logic                                    │
│  - Saga orchestration                               │
│  - Event publishing                                 │
└──────┬──────────────────────────────────────────────┘
       │
       ├──────────────────┬─────────────────┬──────────┐
       ▼                  ▼                 ▼          ▼
┌─────────────┐   ┌─────────────┐   ┌──────────┐   ┌──────────┐
│  Inventory  │   │  Payment    │   │  Kafka   │   │Database/ │
│  Service    │   │  Service    │   │  Events  │   │  Redis   │
└─────────────┘   └─────────────┘   └──────────┘   └──────────┘
```

## Component Details

### 1. Order Service

**Responsibilities**:
- Receive and validate order requests
- Coordinate order lifecycle (saga pattern)
- Publish domain events
- Handle payment results
- Execute compensation logic

**Key Files**:
- `internal/service/order_service.go`: Core order logic
- `internal/service/saga_orchestrator.go`: Saga coordination

### 2. Inventory Service

**Responsibilities**:
- Manage product stock levels
- Reserve inventory atomically
- Release reservations (compensation)
- Commit reservations (finalize)

**Implementation Strategy**:
- **Fast Path**: Redis with Lua scripts for atomic operations
- **Fallback**: PostgreSQL with row-level locking
- **Sync**: Background reconciliation

**Key Files**:
- `internal/service/inventory_client.go`
- `internal/redisclient/scripts/*.lua`

### 3. Payment Service

**Responsibilities**:
- Process payment requests (mocked)
- Publish payment results
- Handle payment timeouts

**Mock Behavior**:
- 90% success rate (configurable)
- Random processing delay (100-500ms)
- Generates unique transaction IDs

**Key Files**:
- `internal/service/payment_service.go`

## Data Flow

### Order Creation Flow

```
1. Client → POST /orders
2. Order Service validates request
3. Order Service creates order (status: CREATED)
4. Order Service creates order items
5. Order Service → Inventory Service: Reserve stock
   ├─ Redis: Atomic decrement (Lua script)
   └─ PostgreSQL: Persistent record
6. If reservation succeeds:
   - Update order status → RESERVED
   - Publish OrderReserved event
7. Payment Service consumes OrderReserved event
8. Payment Service processes payment (mock)
9. Payment Service publishes result:
   ├─ PaymentSuccess → Order confirmed
   └─ PaymentFailed → Compensation triggered
```

### Compensation Flow (Payment Failed)

```
1. Order Service consumes PaymentFailed event
2. Check idempotency (avoid duplicate compensation)
3. Retrieve order items
4. For each item:
   - Inventory Service: Release reservation
   - Redis: Restore available count
   - PostgreSQL: Update inventory
5. Update order status → CANCELLED
6. Mark event as processed
```

## Database Schema

### Core Tables

**products**:
- Catalog of available products
- Immutable during order lifecycle

**inventory**:
- Current stock levels
- Columns: `available`, `reserved`
- Updated atomically

**orders**:
- Order metadata
- Status tracking: CREATED → RESERVED → PAID → CONFIRMED
- Idempotency key for duplicate prevention

**order_items**:
- Line items for each order
- Captures price at time of order

**payments**:
- Payment transaction records
- Links to external payment provider

**processed_events**:
- Event deduplication
- Ensures exactly-once processing

## Event-Driven Architecture

### Event Types

1. **OrderCreated**: Order accepted into system
2. **OrderReserved**: Inventory successfully reserved
3. **OrderPaid**: Payment completed successfully
4. **OrderConfirmed**: Order fully completed
5. **OrderCancelled**: Order cancelled (compensation)
6. **PaymentSuccess**: Payment approved
7. **PaymentFailed**: Payment declined

### Event Structure

```go
type BaseEvent struct {
    EventID   string    // Unique event identifier
    EventType string    // Event type constant
    Timestamp time.Time // Event creation time
}

type PaymentSuccessEvent struct {
    BaseEvent
    OrderID   int64
    PaymentID int64
    Amount    int64
    TxID      string  // External transaction ID
}
```

### Event Flow

```
Order Service ──► Kafka ──► Payment Service
      ▲                            │
      │                            ▼
      └────────── Kafka ◄─────────┘
         (Payment result)
```

## Concurrency Control

### Redis Atomic Operations

**Reserve Stock Lua Script**:
```lua
local available = tonumber(redis.call("HGET", KEYS[1], "available") or "0")
if available >= tonumber(ARGV[1]) then
  redis.call("HINCRBY", KEYS[1], "available", -ARGV[1])
  redis.call("HINCRBY", KEYS[1], "reserved", ARGV[1])
  return 1
end
return 0
```

**Benefits**:
- ✅ Atomic execution (no race conditions)
- ✅ High throughput (microsecond latency)
- ✅ Simple rollback logic

### Database Locking

**PostgreSQL Transaction**:
```sql
BEGIN;
SELECT available FROM inventory 
WHERE product_id = $1 
FOR UPDATE;  -- Row-level lock

UPDATE inventory 
SET available = available - $qty,
    reserved = reserved + $qty
WHERE product_id = $1;
COMMIT;
```

**Benefits**:
- ✅ ACID guarantees
- ✅ No external dependencies
- ⚠️ Lower throughput under high contention

### Hybrid Approach

1. **Try Redis first** (fast path)
2. **Fallback to PostgreSQL** on Redis failure
3. **Async sync** to keep systems consistent
4. **Periodic reconciliation** for drift correction

## Saga Pattern Implementation

### Choreography-based Saga

Each service publishes events and reacts to events from other services.

**Advantages**:
- Loose coupling
- No single point of failure
- Services evolve independently

**Trade-offs**:
- Complex to trace
- Requires robust event infrastructure

### Compensation Logic

**Triggers**:
- Payment failure
- Timeout
- External service error

**Actions**:
1. Identify affected resources
2. Execute reverse operations
3. Update order status
4. Log compensation event

## Observability

### Metrics (Prometheus)

**Business Metrics**:
- `orders_created_total`
- `orders_paid_total`
- `orders_failed_total{reason}`
- `payment_success_rate`

**Technical Metrics**:
- `http_request_duration_seconds`
- `inventory_reserve_latency_seconds`
- `kafka_consumer_lag`

### Tracing (Jaeger)

**Span Hierarchy**:
```
OrderService.CreateOrder
├─ ValidateOrderItems
├─ InventoryClient.ReserveStock
│  ├─ Redis.ReserveStock
│  └─ Store.ReserveStockTx
└─ EventPublisher.PublishOrderReserved
```

### Logging (Zap)

**Structured Logs**:
```json
{
  "level": "info",
  "ts": 1700000000,
  "caller": "service/order_service.go:123",
  "msg": "Order created",
  "order_id": 1001,
  "user_id": 123,
  "trace_id": "abc123"
}
```

## Scalability Considerations

### Horizontal Scaling

- **Stateless services**: Scale order service pods
- **Database**: Read replicas for queries
- **Redis**: Cluster mode for higher throughput
- **Kafka**: Increase partitions

### Performance Optimizations

1. **Connection pooling**: Database & Redis
2. **Batch operations**: Bulk inventory updates
3. **Caching**: Product catalog (rarely changes)
4. **Async processing**: Event publishing non-blocking

### Bottleneck Analysis

**Expected bottlenecks** (in order):
1. Database transactions (inventory locking)
2. Kafka throughput (event publishing)
3. Payment service latency
4. Network I/O

## Security

### Input Validation

- JSON schema validation
- SQL injection prevention (parameterized queries)
- Request size limits

### Idempotency

- Client-provided idempotency keys
- Database unique constraints
- Event deduplication table

### Rate Limiting

- Should be implemented at API Gateway
- Per-user or per-IP limits
- Sliding window algorithm recommended

## Deployment Architecture

### Container Orchestration (K8s)

```yaml
- order-service (3 replicas)
  ├─ Resource limits: 512Mi RAM, 500m CPU
  ├─ Liveness probe: /health
  └─ Readiness probe: /ready

- postgresql (StatefulSet)
  └─ Persistent volume: 50Gi

- redis (StatefulSet)
  └─ Persistent volume: 10Gi

- kafka (StatefulSet, 3 replicas)
  └─ Persistent volume: 20Gi per broker
```

### Network Topology

```
LoadBalancer → Ingress → order-service pods
                          ↓
                    Internal Services
                    ├─ PostgreSQL
                    ├─ Redis
                    └─ Kafka
```

## Failure Modes & Recovery

### Database Failure

- **Impact**: No new orders accepted
- **Recovery**: Automatic failover to replica
- **Mitigation**: Health checks, circuit breaker

### Redis Failure

- **Impact**: Slower inventory operations
- **Recovery**: Automatic fallback to PostgreSQL
- **Mitigation**: Redis sentinel for HA

### Kafka Failure

- **Impact**: Events queued, payment delayed
- **Recovery**: Kafka auto-recovery
- **Mitigation**: Multiple brokers, replication

### Payment Service Failure

- **Impact**: Orders stuck in RESERVED state
- **Recovery**: Timeout + compensation
- **Mitigation**: Retry with exponential backoff

## Future Enhancements

1. **Circuit Breaker**: Prevent cascade failures
2. **API Rate Limiting**: Protect against abuse
3. **Read Replicas**: Scale read operations
4. **Event Sourcing**: Complete audit trail
5. **CQRS**: Separate read/write models
6. **GraphQL API**: Flexible querying
7. **gRPC**: Inter-service communication
8. **Multi-region**: Geographic distribution
