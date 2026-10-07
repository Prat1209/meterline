# Meterline

A billing backend for an AI API: it records metered token usage, bills customers through Stripe (test mode), and guarantees no request is ever charged twice.

> Work in progress. See `PROJECT_BRIEF.md` for scope and `BUILD_LOG.md` for current status.

## Run locally

Requires Go 1.24+ and Docker.

```sh
docker compose up -d postgres
export DATABASE_URL="postgres://meterline:meterline@localhost:5432/meterline?sslmode=disable"
go run ./cmd/seed "Acme AI"   # applies migrations, creates an account, prints its API key once
go run ./cmd/api              # applies migrations on startup, listens on :8080
```

PowerShell: set the variable with `$env:DATABASE_URL="postgres://..."` instead of `export`.

The service reads configuration from environment variables only (see `.env.example`).

## Record usage

```sh
curl -i -X POST localhost:8080/v1/usage \
  -H "Authorization: Bearer mk_test_..." \
  -H "Idempotency-Key: req-123" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-x","input_tokens":1200,"output_tokens":340}'
```

| Case | Response |
|---|---|
| First request | `201` with the created event |
| Same key, same body | the same `201` body again, with `Idempotent-Replayed: true`; no new row |
| Same key, different body | `422 idempotency_key_reused` |
| Missing key or invalid body | `400` |
| Missing or invalid API key | `401` |

Idempotency keys are scoped per account. The claim, the usage insert and the stored response happen in one transaction, and concurrent requests with the same key serialize on the key's primary-key lock. The test suite fires 100 concurrent duplicates and asserts exactly one row.

## Test

```sh
go test -race ./...
```

Integration tests need `DATABASE_URL` pointing at a running Postgres; without it they are skipped locally (CI always runs them).
