.PHONY: run test lint db-up db-down

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
