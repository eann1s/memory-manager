.PHONY: test test-integration db-start db-migrate db-reset help

help:
	@echo "Available targets:"
	@echo "  make db-start          - Start PostgreSQL database using docker-compose"
	@echo "  make db-migrate        - Run database migrations"
	@echo "  make db-reset          - Drop all data and re-run migrations"
	@echo "  make test              - Run all tests"
	@echo "  make test-integration  - Run integration tests only"

db-start:
	docker-compose -f docker-compose.local.yml up -d db

db-migrate:
	go run ./cmd/migrate up

db-reset:
	docker-compose -f docker-compose.local.yml down -v
	docker-compose -f docker-compose.local.yml up -d db
	sleep 3
	go run ./cmd/migrate up

test:
	go test -v ./...

test-integration:
	go test -v ./internal/store -run "^Test"
