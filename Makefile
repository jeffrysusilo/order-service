.PHONY: help build run test clean docker-up docker-down migrate seed

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application
	go build -o bin/server ./cmd/server

run: ## Run the application locally
	go run ./cmd/server/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

deps: ## Download dependencies
	go mod download
	go mod tidy

docker-up: ## Start all services with docker-compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## Show docker logs
	docker-compose logs -f order-service

docker-build: ## Build docker image
	docker-compose build

docker-rebuild: docker-down docker-build docker-up ## Rebuild and restart services

migrate: ## Run database migrations
	@echo "Running migrations..."
	@docker exec -i order-postgres psql -U app -d app < migrations/001_init_schema.sql

seed: ## Seed database with sample data
	@echo "Seeding database..."
	@docker exec -i order-postgres psql -U app -d app < migrations/002_seed_data.sql

db-reset: docker-down ## Reset database (warning: deletes all data)
	docker volume rm order-service_postgres_data || true
	$(MAKE) docker-up
	sleep 5
	$(MAKE) migrate
	$(MAKE) seed

load-test: ## Run k6 load test
	k6 run tests/load/order_test.js

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

lint: ## Run linter
	golangci-lint run

.DEFAULT_GOAL := help
