.PHONY: test test-integration test-openai db-start db-migrate db-reset help

help:
	@echo "Available targets:"
	@echo "  make db-start          - Start PostgreSQL database using docker-compose"
	@echo "  make db-migrate        - Run database migrations"
	@echo "  make db-reset          - Drop all data and re-run migrations"
	@echo "  make test              - Run all tests"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-openai       - Run OpenAI integration tests (requires OPENAI_API_KEY)"

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

test-openai:
	RUN_OPENAI_INTEGRATION=1 go test -v ./internal/openai -run TestEmbed_Integration
