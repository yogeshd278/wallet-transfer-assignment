# Pull request: Wallet Transfer Service

> Copy this into your PR description when opening from `feature/*` → `main`.

## Summary

<!-- 2–4 bullets: what the PR delivers -->

- Implements `POST /transfers` with client idempotency and `GET /transfers/:id`
- PostgreSQL schema: `wallets`, `transfers`, `ledger_entries`, `idempotency_records`
- Double-entry ledger on successful transfers; state machine `PENDING` → `PROCESSED` | `FAILED`
- Integration tests including idempotency and concurrent requests

## Schema

<!-- Link to migration files; summarize tables and key constraints -->

| Table | Purpose | Key constraints / indexes |
|-------|---------|---------------------------|
| `wallets` | Cached balance per wallet | `balance >= 0`, PK `id` |
| `transfers` | Transfer lifecycle | `UNIQUE(idempotency_key)`, status CHECK |
| `ledger_entries` | Audit trail | 2 rows per PROCESSED transfer; DEBIT/CREDIT CHECK |
| `idempotency_records` | Exactly-once replay | PK `idempotency_key`, stores response snapshot |

**Migration path:** `migrations/000001_*.up.sql` — apply via `make migrate-up`.

## Idempotency

<!-- How exactly-once is achieved -->

- Client supplies `idempotencyKey` (header and/or body — document which).
- First request: single DB transaction claims key, processes transfer, persists response in `idempotency_records`.
- Duplicate request: unique constraint on key → load completed record → return **original** JSON without wallet mutation.
- Same key + different payload fingerprint → `409 Conflict`.

## Concurrency

<!-- Justify choices -->

- **Isolation:** `READ COMMITTED` with explicit `SELECT ... FOR UPDATE` on wallet rows (source + destination).
- **Lock ordering:** wallets locked in sorted ID order to reduce deadlocks.
- **Alternative considered:** optimistic locking on `wallets.version` — rejected/more retries under contention (or: chosen because …).

## Tradeoffs & limitations

<!-- Honest scope -->

- Wallet `balance` is denormalized for fast reads; ledger is source of truth for audit.
- No authentication in v1 / no distributed transactions across services.
- Idempotency records retained indefinitely (or: TTL X days) — cleanup job not implemented.

## Test plan

- [ ] `go test ./... -race`
- [ ] Integration: happy path transfer, balances + 2 ledger lines
- [ ] Idempotency: 20 parallel POSTs, same key → single debit
- [ ] Insufficient funds → no ledger, balance unchanged
- [ ] Concurrent transfers from same wallet → one success, one failure
- [ ] Manual: `curl` examples from README

## How to run

```bash
docker compose up -d
cp .env.example .env
make migrate-up
go run ./cmd/server
```

## AI usage disclosure

<!-- Assignment requires addressing all three points -->

1. **What I used AI for:**  
   <!-- e.g. scaffolding Gin handlers, drafting SQL migrations, brainstorming test cases, wording PR sections -->

2. **What I verified myself:**  
   <!-- e.g. transaction boundaries, ledger invariants, ran integration tests, reviewed locking order -->

3. **What I would do differently without AI:**  
   <!-- e.g. spend more time on hand-written concurrency tests first; deeper read of PG docs on SKIP LOCKED -->
