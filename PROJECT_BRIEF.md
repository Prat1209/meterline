# Project Brief — Usage-Based Billing Service (working name: TBD)

> **Stable reference.** This file changes only when the scope changes on purpose.
> For current status, decisions and next steps, see `BUILD_LOG.md`. **If the two disagree, `BUILD_LOG.md` wins.**

## Why this exists
This is a portfolio project for my Stripe Software Engineer, New Grad application (referred, Oct 2026).

The goal is to show production-minded backend engineering in Stripe's domain: reliable APIs, payments correctness, and risk. A recruiter should understand it in 10 seconds, and an engineer should be able to dig into it for 30 minutes.

## What it does (one sentence)
It's a billing backend for an AI API: it records metered token usage, bills customers through Stripe (test mode), and guarantees no request is ever charged twice.

## Scope: must have
1. **Usage ingestion API with idempotency keys.** A replayed request returns the same result and never creates a duplicate record.
2. **Stripe integration (test mode only).** Customers, usage-based prices, usage reporting, and invoices.
3. **Webhook consumer.** Verifies signatures, retries with backoff, and dedupes by Stripe event ID.
4. **Double-entry ledger in Postgres.** Every money movement is a balanced debit and credit.
5. **Reconciliation job.** Compares the ledger to Stripe and records any mismatches.
6. **Usage anomaly / fraud score.** It starts as a z-score on usage velocity per customer.
7. **Tests and CI** on every PR.
8. **Deployed** with a live link.

## Nice to have (only after the must-haves ship)
- A minimal dashboard showing usage, invoices, and flagged anomalies
- A load test with published numbers
- A feature-flagged change plus a zero-downtime migration, written up

## Non-goals
- No real money and no live keys. Stripe test mode only.
- No UI that copies Stripe's look or branding.
- No LLM chat interface (LNG Ops Pulse already covers that).
- No auth system beyond simple API keys for the demo.

## Stack
| Layer | Choice |
|---|---|
| Backend language | **OPEN**: Go or TypeScript (see BUILD_LOG decisions) |
| Database | Postgres |
| Payments | Stripe API, test mode |
| Hosting | **OPEN** |
| CI | GitHub Actions |

## Architecture (target)
```
Client ──► Usage API ──► idempotency check ──► Postgres
            │                                  (usage_events, idempotency_keys,
            │                                   accounts, ledger_entries)
            └──► report usage ──► Stripe Billing (test mode)

Stripe ──► Webhook endpoint ──► verify signature ──► dedupe by event ID ──► ledger

Scheduled reconciliation job ──► compare ledger vs Stripe ──► mismatches table
Anomaly scorer ──► flag usage spikes per customer
```

## Conventions
- Store money as integer minor units (cents), never as floats.
- Every write that touches money happens inside a single DB transaction.
- Every external call has a timeout and a retry policy with backoff.
- Secrets come from env vars only. `.env.example` is committed; `.env` never is.
- Work on branches and merge via PR with CI green. Each PR description says what changed and why.
- Any AI-written code gets read and tested before merge. Notable catches go in BUILD_LOG under "AI catches".

## Definition of done (whole project)
- The live link works. The README has a one-line hook, a GIF, and setup steps.
- `DESIGN.md` (1–2 pages) covers decisions, tradeoffs, and failure modes.
- Headline numbers are measured and in the README, e.g. duplicates across N replayed requests and p95 latency.
- The README has a "How I used AI" section.
