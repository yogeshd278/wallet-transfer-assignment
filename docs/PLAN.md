# Master implementation plan

## Goals (assignment)

Deliver a small HTTP service that:

1. Accepts `POST /transfers` with an `idempotencyKey` and performs wallet debits/credits **at most once** per key.
2. Persists **transfers** with lifecycle `PENDING` → `PROCESSED` | `FAILED`.
3. Maintains a **double-entry ledger** (`ledger_entries`) tied to each processed transfer.
4. Uses **PostgreSQL**, **layered architecture**, and **automated tests** (unit + integration + concurrency).
5. Documents tradeoffs in the PR (schema, idempotency, concurrency, AI usage).

## Non-goals (v1)

- Authentication / multi-tenant API keys (optional stretch).
- Integration with `wallet_service_be` or order settlement.
- Async outbox / message queues (stretch).
- Admin APIs beyond `GET /transfers/:id` (stretch).

## Recommended stack decision

| Choice | Rationale |
|--------|-----------|
| **Go + Gin** | Matches `*_service_be` services; fast to test; good concurrency story |
| **PostgreSQL** | Assignment requirement; row-level locking fits wallet updates |
| **pgx + sqlc or hand-written repo SQL** | Ledger invariants are easier to audit than ORM magic |
| **testcontainers-go** or **docker-compose** for CI integration tests | Real locking/isolation behavior |

**Alternative:** Rust/Axum + SQLx — stronger type safety, but inconsistent with monorepo and slower for reviewers familiar with Go.

## Phases

### Phase 0 — Bootstrap (0.5–1 day) — done

- [x] Initialize Go module `wallet_transfer_service` (or `github.com/powerxchange/wallet-transfer-service`)
- [x] `docker-compose.yml`: Postgres 15, healthcheck, published port 5433
- [x] `.env.example`, `Makefile` (`run`, `test`, `migrate-up`, `migrate-down`)
- [x] `cmd/server`: health `GET /health`, graceful shutdown
- [x] Structured logging (`slog`) + request ID middleware
- [x] Copy/adapt PR template from `docs/PR_TEMPLATE.md` into `.github/pull_request_template.md` if using GitHub; else keep in `docs/`

**Exit criteria:** `curl /health` OK, migrations apply cleanly on empty DB. (`go build` / `go test` verified; run `make docker-up && make migrate-up` locally to confirm DB exit criteria.)

### Phase 1 — Schema & domain (1 day) — done

- [x] Migrations: `wallets`, `transfers`, `ledger_entries`, `idempotency_records` (see DESIGN)
- [x] Domain types: `Money` (int64 minor units), `TransferStatus`, validation errors
- [x] Seed script or migration for demo wallets (optional)
- [x] Repository interfaces in `internal/domain` or `internal/repository` ports

**Exit criteria:** Migration up/down idempotent; domain unit tests for money/rules.

### Phase 2 — Core transfer use case (1.5–2 days)

- [x] `POST /transfers` handler: bind JSON, validate, call service
- [x] Service: single DB transaction boundary:
  - claim idempotency key
  - insert `PENDING` transfer
  - lock wallets, check balance, update balances
  - insert ledger pair, mark `PROCESSED`
- [x] `GET /transfers/:id` (read model for clients)
- [x] Map domain errors → HTTP (409 insufficient funds, 422 validation, 404)

**Exit criteria:** Happy-path transfer via curl; ledger sums to zero per transfer.

### Phase 3 — Idempotency (1 day)

- [x] `idempotency_records` lifecycle: `IN_PROGRESS` → `COMPLETED` (store response body + status code)
- [x] Duplicate `POST` with same key: return **stored response** without re-debiting
- [x] Concurrent duplicate requests: one wins via unique constraint + retry read
- [x] Hash or canonical JSON of request body for optional conflict detection (same key, different body → 409)

**Exit criteria:** Integration test fires 20 parallel POSTs with same key → one transfer, identical responses.

### Phase 4 — Failure paths & consistency (0.5–1 day)

- [x] Insufficient balance → `FAILED` transfer, no ledger rows (or compensating policy documented)
- [x] DB failure mid-transaction → full rollback, idempotency record cleaned or left `IN_PROGRESS` with timeout policy
- [x] Invalid wallet IDs → `FAILED` or 404 before side effects

**Exit criteria:** Tests assert no orphan ledger rows; balance unchanged on failure.

### Phase 5 — Concurrency hardening (1 day)

- [x] `SELECT ... FOR UPDATE` on source (and optionally dest) wallet rows
- [x] Document isolation level (`READ COMMITTED` default + explicit locks vs `SERIALIZABLE`)
- [x] Stress test: two transfers from same wallet concurrently — one succeeds, one fails cleanly

**Exit criteria:** `go test -race` passes; concurrency integration test green.

### Phase 6 — Observability & polish (0.5 day)

- [ ] Metrics hooks (optional): transfer count, latency histogram
- [x] README runbook: migrations, env vars, example curls
- [ ] OpenAPI snippet or `api/openapi.yaml` (stretch)

### Phase 7 — PR & review (0.5 day)

- [x] Fill `docs/PR_TEMPLATE.md` in actual PR description
- [x] Self-review checklist below
- [x] AI usage section (3 required points)

## Time estimate summary

| Phase | Estimate |
|-------|----------|
| 0 Bootstrap | 0.5–1 d |
| 1 Schema/domain | 1 d |
| 2 Core transfer | 1.5–2 d |
| 3 Idempotency | 1 d |
| 4 Failures | 0.5–1 d |
| 5 Concurrency | 1 d |
| 6 Polish | 0.5 d |
| 7 PR | 0.5 d |
| **Total** | **~6–8 days** (focused) |

## Self-review checklist (pre-PR)

- [x] All money in **minor units** (`BIGINT`), never `float`
- [x] Every `PROCESSED` transfer has exactly **two** ledger entries (debit/credit) summing to 0
- [x] `idempotencyKey` unique per client scope (define: global unique vs per-tenant if auth added)
- [x] Transaction wraps: idempotency claim + transfer + wallets + ledger
- [x] Tests: happy path, idempotency, insufficient funds, invalid wallets, concurrent duplicates, concurrent same-wallet
- [x] No secrets in repo; `.env` gitignored

## Stretch goals (optional)

- [ ] `GET /wallets/:id` balance + ledger tail
- [ ] Outbox table + worker for async notifications
- [ ] API key middleware + idempotency scoped to `client_id`
- [ ] golangci-lint + GitHub Actions
- [ ] Explain migration from float `wallet_service_be.balance` to ledger-first model

## Milestone order (critical path)

```mermaid
flowchart LR
  P0[Bootstrap] --> P1[Schema]
  P1 --> P2[Transfer TX]
  P2 --> P3[Idempotency]
  P3 --> P4[Failures]
  P4 --> P5[Concurrency]
  P5 --> P6[Docs/PR]
```

Idempotency (Phase 3) should not be deferred after concurrency tests — duplicate POSTs are the highest-risk bug class.
