# Deployment Guide

## Prerequisites

- Docker & Docker Compose installed
- At least 4GB RAM available
- Ports available: 8080, 5432, 6379, 9092, 16686, 9090, 3000

## Quick Start

1. **Clone the repository**
```bash
git clone <repository-url>
cd order-service
```

2. **Start services**
```bash
docker-compose up -d
```

3. **Check service health**
```bash
# Wait for services to be ready (may take 30-60 seconds)
docker-compose ps

# Check order service health
curl http://localhost:8080/health
```

4. **Run migrations**
```bash
make migrate
make seed
```

## Production Deployment

### Environment Variables

Create `.env` file with production values:

```bash
# Server
PORT=8080
ENV=production

# Database (use secure credentials)
DATABASE_URL=postgres://username:password@postgres-host:5432/dbname?sslmode=require

# Redis
REDIS_ADDR=redis-host:6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0

# Kafka
KAFKA_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
KAFKA_TOPIC_ORDER_EVENTS=order-events
KAFKA_CONSUMER_GROUP=order-service-group

# Observability
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
PROMETHEUS_PORT=9090
```

### Kubernetes Deployment

Example Kubernetes deployment files:

**Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
    spec:
      containers:
      - name: order-service
        image: order-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: order-service-secrets
              key: database-url
        - name: REDIS_ADDR
          value: "redis-service:6379"
        - name: KAFKA_BROKERS
          value: "kafka-service:9092"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

**Service**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: order-service
spec:
  selector:
    app: order-service
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Scaling

**Horizontal scaling**:
```bash
# Docker Compose
docker-compose up -d --scale order-service=3

# Kubernetes
kubectl scale deployment order-service --replicas=5
```

### Monitoring

Access monitoring tools:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

Import Grafana dashboards for:
- Go metrics
- HTTP request metrics
- Kafka consumer lag

## Troubleshooting

### Service won't start

Check logs:
```bash
docker-compose logs order-service
```

### Database connection failed

Verify PostgreSQL is ready:
```bash
docker exec -it order-postgres psql -U app -d app -c "SELECT 1"
```

### Kafka connection issues

Check Kafka broker:
```bash
docker exec order-kafka kafka-topics --list --bootstrap-server localhost:9092
```

### High memory usage

Adjust Go memory settings:
```bash
GOMEMLIMIT=256MiB ./server
```

## Backup & Recovery

### Database Backup

```bash
docker exec order-postgres pg_dump -U app app > backup.sql
```

### Restore

```bash
docker exec -i order-postgres psql -U app app < backup.sql
```
