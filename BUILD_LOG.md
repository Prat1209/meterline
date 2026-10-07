# Build Log

> **Living state of the project.** Update this at the end of every working session, then replace the copy in Project knowledge. Delete the old one first so there's only ever one version.
> Newest entries go on top.

## Current status
- **Phase:** M0, setup
- **Last updated:** 2026-10-07
- **Live link:** none yet
- **Repo:** none yet

## Milestones (~2 weeks total)
- [ ] **M0:** Lock decisions (language, name, hosting), create the repo, set up a CI skeleton
- [ ] **M1:** Usage API, idempotency keys, Postgres schema
- [ ] **M2:** Stripe test-mode integration (customers, prices, usage reporting, invoices)
- [ ] **M3:** Webhooks (verify, retry, dedupe) and the double-entry ledger
- [ ] **M4:** Reconciliation job
- [ ] **M5:** Anomaly score
- [ ] **M6:** Deploy, measure headline numbers, write README and DESIGN.md
- [ ] *(Nice to have)* Dashboard, load test

## Next up
1. Decide the backend language (Go vs TypeScript)
2. Pick a project name
3. Create the GitHub repo and add these two files to its root

## Open questions
- Where will the API and Postgres be hosted?
- Should the anomaly score run inline on ingest, or as a batch job?

## Decisions (newest first)
| Date | Decision | Why | Alternatives rejected |
|---|---|---|---|
| 2026-10-07 | Build on Stripe test mode: usage-based billing for an AI API | Covers the JD's main themes: payments infra, reliable APIs, risk, AI | Ledger-only project; fraud-only ML project |

## Repo map
_Paste the `tree -L 2` output here once the repo exists, and keep it current._

## Schema / API contract
_Keep the current tables and endpoints listed here, in sync with the code._

## Known issues / tech debt
_None yet._

## AI catches (for the README's "How I used AI" section)
| Date | What the AI suggested | What was wrong | Fix |
|---|---|---|---|

## Session notes (newest first)
### 2026-10-07
- Created the project, wrote PROJECT_BRIEF.md and BUILD_LOG.md.
