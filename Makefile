-include .env
export

DATABASE_URL ?= postgres://wallet:wallet@localhost:5433/wallet_transfer?sslmode=disable
MIGRATE ?= migrate
MIGRATION_PATH ?= migrations

.PHONY: build run test test-integration migrate-up migrate-down migrate-install docker-up docker-down

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./... -count=1

test-integration:
	go test ./tests/integration/... -tags=integration -race -count=1

migrate-install:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up:
	$(MIGRATE) -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" down 1

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down
