# CLAUDE.md

Working notes for AI pair-programming sessions on Meterline. Read this first, then `BUILD_LOG.md`.

## What this is
A portfolio billing backend for an AI API (Stripe SWE New Grad application). It records metered token usage, bills through Stripe **test mode only**, and guarantees no request is charged twice.

- `PROJECT_BRIEF.md`: stable scope, stack, conventions, non-goals.
- `BUILD_LOG.md`: current status, milestones, decisions, next steps. **If the two disagree, BUILD_LOG.md wins.**

## How to work
- Start by reading "Current status" and "Next up" in `BUILD_LOG.md` and confirm which milestone is active.
- Stay inside the current milestone. Anything for a later milestone or a non-goal goes into "Next up", not into code.
- Never silently change a past decision. Propose a new row for the Decisions table instead.
- Discuss a design before building it for each new milestone. Prat often says "discuss mode" for this.
- Work on a branch, open a PR, get CI green. Prat reviews and merges, then runs `git pull` locally.
- PR descriptions say what changed and why (Before / After / How).
- Notable AI mistakes go in BUILD_LOG.md under "AI catches".
- On "wrap up": produce the full updated `BUILD_LOG.md` (status, milestones, next up, decisions, AI catches, dated session note).

## Conventions (non-negotiable)
- Money is integer minor units (cents, `BIGINT`). Never floats.
- Every write that touches money happens inside one DB transaction.
- Every external call (Stripe, etc.) has a timeout and a retry policy with backoff; POSTs to Stripe carry an idempotency key.
- Secrets come from env vars only. `.env.example` is committed, `.env` never is. Config rejects `sk_live_` keys.
- Errors use one JSON shape via `internal/httperr`: `{"error":{"type","message"}}`.
- Each request has a DB timeout (`context.WithTimeout`); never use `context.Background()` in a handler.

## Commands
```sh
docker compose up -d postgres
export DATABASE_URL="postgres://meterline:meterline@localhost:5432/meterline?sslmode=disable"
go run ./cmd/seed "Acme AI"     # migrate + create an account, prints the API key once
go run ./cmd/api                # migrate on startup, serve on :8080
go run ./cmd/migrate            # migrations only
go vet ./... && go test -race ./...
golangci-lint run ./...         # CI runs this; run it before pushing
```
Integration tests need `DATABASE_URL`. They skip locally without it and fail in CI without it.

## Code map
- `cmd/api` wiring and graceful shutdown; `cmd/seed`, `cmd/migrate` small CLIs.
- `internal/config` env config. `internal/db` pgx pool with ping retry, goose migrations.
- `internal/auth` API keys (`mk_test_` + random, stored as sha256) and Bearer middleware.
- `internal/usage` `POST /v1/usage`: `request.go` (validate, canonical hash), `store.go` (the idempotent transaction), `handler.go`.
- `migrations/` numbered goose SQL files, embedded by `migrations.go`. Add a new file for every schema change; never edit an applied migration.

## Idempotency design (keep it intact)
One transaction: `INSERT INTO idempotency_keys ... ON CONFLICT DO NOTHING`. If it inserted, write the event and store the exact response bytes (`BYTEA`). If it conflicted, compare `request_hash`: same means replay the stored response with `Idempotent-Replayed: true`, different means 422. Concurrent duplicates serialize on the key's primary-key lock. The test `TestConcurrentDuplicatesCreateOneRow` (100 concurrent requests, 1 row) must keep passing.

## Environment gotchas
- Prat is on Windows with PowerShell. Give commands one per code block. `$env:VAR=...` only lasts for the current window. `make` is usually not installed. Use `Invoke-WebRequest -UseBasicParsing` (or `curl.exe`) for HTTP calls.
- `go test -race` needs cgo on Windows; locally use `go test ./...`, CI runs `-race`.
- `go.mod` targets Go 1.24. When adding dependencies, pin versions that don't bump the `go` directive.
- `.gitattributes` forces LF line endings.
