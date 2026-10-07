# Meterline

A billing backend for an AI API: it records metered token usage, bills customers through Stripe (test mode), and guarantees no request is ever charged twice.

> Work in progress. See `PROJECT_BRIEF.md` for scope and `BUILD_LOG.md` for current status.

## Run locally

Requires Go 1.24+ and Docker.

```sh
cp .env.example .env        # then edit values
docker compose up -d postgres
go run ./cmd/api            # reads env vars; load .env first (see below)
curl localhost:8080/healthz
```

The service reads configuration from environment variables only. Load `.env` into your shell before running (for example `set -a; . ./.env; set +a` in bash, or your editor's env-file support).

## Test

```sh
go test -race ./...
```
