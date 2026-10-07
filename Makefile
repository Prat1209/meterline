.PHONY: run test lint db-up db-down migrate seed

run:
	go run ./cmd/api

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed
