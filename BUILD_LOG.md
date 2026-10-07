# Build Log

> **Living state of the project.** Update this at the end of every working session, then replace the copy in Project knowledge. Delete the old one first so there's only ever one version.
> Newest entries go on top.

## Current status
- **Phase:** M1 done, M2 (Stripe test-mode integration) next
- **Last updated:** 2026-10-07
- **Live link:** none yet (hosting decided: Render, deploy in M6)
- **Repo:** https://github.com/Prat1209/meterline (public, CI green on `main`)
- **Local checkout:** `C:\Users\prath\Documents\meterline`

## Milestones (~2 weeks total)
- [x] **M0:** Lock decisions (language, name, hosting), create the repo, set up a CI skeleton. *Done 2026-10-07.*
- [x] **M1:** Usage API, idempotency keys, Postgres schema. *Done 2026-10-07, PR #1.*
- [ ] **M2:** Stripe test-mode integration (customers, prices, usage reporting, invoices)
- [ ] **M3:** Webhooks (verify, retry, dedupe) and the double-entry ledger
- [ ] **M4:** Reconciliation job
- [ ] **M5:** Anomaly score
- [ ] **M6:** Deploy, measure headline numbers, write README and DESIGN.md
- [ ] *(Nice to have)* Dashboard, load test

## Next up
1. **M2 design (discuss first):** how accounts map to Stripe customers, how prices are modelled (integer cents per 1K tokens, input vs output), and when usage gets reported to Stripe (inline vs batched).
2. Create a Stripe test-mode account and put `STRIPE_SECRET_KEY=sk_test_...` in a local `.env` (never committed). The service already refuses `sk_live_` keys.
3. Stripe client wrapper with timeouts, retries with backoff, and Stripe idempotency keys on every POST.
4. Tech debt from M1 (small, can ride along with M2):
   - Default `occurred_at` from the DB clock instead of the Go process clock (see Known issues).
   - Expire idempotency keys after 24h like Stripe does (cleanup job or `created_at` filter).

## Open questions
- Should the anomaly score run inline on ingest, or as a batch job? (M5)
- Report usage to Stripe per event, or aggregate and report on a schedule? (M2)

## Decisions (newest first)
| Date | Decision | Why | Alternatives rejected |
|---|---|---|---|
| 2026-10-07 | Integration tests run against real Postgres; skip locally without `DATABASE_URL`, fail in CI without it | Idempotency under concurrency can only be proven against a real DB; CI can never silently skip them | Mocks / SQLite (can't reproduce Postgres locking) |
| 2026-10-07 | API keys are `mk_test_` + 32 random bytes, stored only as sha256 | Raw keys never at rest; simple enough for the demo (brief: no auth system beyond API keys) | Storing raw keys; JWT/OAuth (non-goal) |
| 2026-10-07 | pgx/v5 + goose (SQL migrations embedded in the binary, applied on startup) | Explicit SQL and transactions are part of what the project shows; one binary to deploy | ORM (hides the transactions); golang-migrate CLI (extra tool to install) |
| 2026-10-07 | Stored idempotent response is `BYTEA`, not `JSONB` | Postgres normalizes JSONB key order/whitespace, so replays would not be byte-identical | `JSONB` (original design) |
| 2026-10-07 | Idempotency = one transaction: claim key with `INSERT ... ON CONFLICT DO NOTHING`, then insert event and store response | Concurrent duplicates block on the key's primary-key lock until the first commits, so no in-progress state or advisory locks are needed; any error rolls back the key too | Advisory locks; "in progress" status row; check-then-insert (racy) |
| 2026-10-07 | Idempotency keys scoped per account; same key + different body returns 422 | Matches Stripe semantics; catches client bugs instead of silently replaying the wrong result | Global keys; replaying regardless of body |
| 2026-10-07 | M1 stores token counts only, no money | Prices don't exist until M2; avoids inventing cents too early | Storing a cost column now |
| 2026-10-07 | Hosting: Render (managed Postgres) | Simplest deploy path; only needed in M6 | Fly.io |
| 2026-10-07 | Name: Meterline ("okay for now") | Short, describes metering; easy to rename before launch | Tallyhook, Ledgerly |
| 2026-10-07 | Backend language: Go (stdlib `net/http`, Go 1.24 in `go.mod`) | Stripe uses a lot of Go; explicit transactions, timeouts and retries are visible in the code | TypeScript (faster to start, weaker signal for this project) |
| 2026-10-07 | Build on Stripe test mode: usage-based billing for an AI API | Covers the JD's main themes: payments infra, reliable APIs, risk, AI | Ledger-only project; fraud-only ML project |

## Repo map
```
.github/workflows/ci.yml   vet, race tests, golangci-lint; Postgres 16 service container
cmd/api/                   HTTP server: config -> DB connect -> migrate -> serve, graceful shutdown
cmd/migrate/               apply migrations and exit
cmd/seed/                  create a demo account, print its API key once
internal/auth/             API-key generation, hashing, Bearer middleware
internal/config/           env-only config, fails fast, rejects sk_live_ keys
internal/db/               pgx pool (connect timeout, ping retry with backoff), goose migrations
internal/httpapi/          router and /healthz
internal/httperr/          one JSON error shape: {"error":{"type","message"}}
internal/usage/            POST /v1/usage: request validation/hashing, idempotent store, handler
migrations/                00001_init.sql (embedded via migrations.go)
docker-compose.yml         local Postgres 16
CLAUDE.md                  working notes for AI pair sessions
PROJECT_BRIEF.md, BUILD_LOG.md
```

## Schema / API contract
**Tables (migration 00001)**
- `accounts(id BIGSERIAL PK, name, api_key_hash BYTEA UNIQUE, created_at)`
- `idempotency_keys(account_id FK, key, request_hash BYTEA, response_status INT, response_body BYTEA, created_at, PK(account_id, key))`
- `usage_events(id BIGSERIAL PK, account_id FK, idempotency_key, model, input_tokens BIGINT >= 0, output_tokens BIGINT >= 0, occurred_at, created_at, UNIQUE(account_id, idempotency_key))`, index on `(account_id, occurred_at)`

**Endpoints**
- `GET /healthz` returns `{"status":"ok"}` (no DB check)
- `POST /v1/usage` with `Authorization: Bearer mk_test_...` and required `Idempotency-Key` (1-255 chars). Body `{"model","input_tokens","output_tokens","occurred_at"?}`.
  - `201` new event; replay returns the identical `201` body plus `Idempotent-Replayed: true`
  - `422 idempotency_key_reused` same key, different body
  - `400 invalid_request` missing key or bad body; `401 unauthorized` bad or missing API key

## Known issues / tech debt
- `occurred_at` defaults to the Go process clock while `created_at` uses Postgres `now()`. With Docker on Windows the clocks differ slightly, so `created_at` can be microseconds earlier than `occurred_at`. Harmless now; fix by defaulting in SQL.
- Idempotency keys never expire (Stripe expires them after 24h).
- No rate limiting on the API.
- `/healthz` doesn't check the DB (fine for liveness; add a readiness check before deploy in M6).
- On Windows, `go test -race` needs cgo/gcc; run plain `go test ./...` locally. CI runs `-race`.

## AI catches (for the README's "How I used AI" section)
| Date | What the AI suggested | What was wrong | Fix |
|---|---|---|---|
| 2026-10-07 | Unchecked `defer sqlDB.Close()` and `defer resp.Body.Close()` | golangci-lint (errcheck) would have failed CI | Ran the linter locally before pushing; explicitly discard the errors with a comment |
| 2026-10-07 | Store the idempotent response as `JSONB` (in the M1 design) | JSONB normalizes key order and whitespace, so replays would not be byte-identical | Switched to `BYTEA` while building; test asserts identical bytes |
| 2026-10-07 | `go get ...@latest` for pgx and goose | Pulled versions requiring Go 1.26 and silently bumped `go.mod`, which would force a newer Go on every machine | Pinned pgx v5.7.5 and goose v3.24.3 to keep `go 1.24` |
| 2026-10-07 | Told Prat that `$env:DATABASE_URL` carries over | It only lasts for the current PowerShell window; the seed failed with "DATABASE_URL is required" in a new window | Set it per window (fail-fast config caught it immediately) |

## Session notes (newest first)
### 2026-10-07 (session 2)
- **M0 finished:** picked Go, Meterline and Render; scaffolded the project in `C:\Users\prath\Documents\meterline`; verified tests, Docker Postgres and `/healthz` locally; created the public repo and pushed. CI green on the first push.
- **M1 finished:** agreed the design in `notes/m1-design.md`, then built `POST /v1/usage` with API-key auth, idempotency keys and migration 00001. 100 concurrent duplicate requests leave exactly 1 row (run 10x with `-race`). PR #1 merged with CI green.
- Verified by hand on Windows: first request returned 201, replay returned the same body with `Idempotent-Replayed: true`, changed body returned 422.
- Connected the repo to Claude: Claude now opens PRs, Prat reviews, merges, and runs `git pull` locally.
- Added `CLAUDE.md` to the repo with commands, conventions and workflow.

### 2026-10-07 (session 1)
- Created the project, wrote PROJECT_BRIEF.md and BUILD_LOG.md.
