# Implementation notes (Phases 2–7)

Summary for PR reviewers: schema usage, idempotency, and concurrency choices implemented in this branch.

## Schema

- **Money:** all amounts are `BIGINT` minor units (paise); wallet `balance` is a cached aggregate, ledger is audit trail.
- **Transfers:** `PENDING` → `PROCESSED` | `FAILED`; `UNIQUE(idempotency_key)` prevents duplicate transfer rows.
- **Ledger:** exactly two rows per `PROCESSED` transfer (`DEBIT` source, `CREDIT` destination); none for `FAILED`.
- **Idempotency:** `idempotency_records` holds `IN_PROGRESS` → `COMPLETED` with stored HTTP status + JSON body for replay.

## Idempotency

1. Single DB transaction per `POST /transfers` attempt wraps: idempotency claim, transfer row, wallet updates, ledger, idempotency completion.
2. **Claim:** `INSERT` idempotency row `IN_PROGRESS`; on `23505` unique violation, `SELECT … FOR UPDATE` existing row.
3. **Replay:** if status is `COMPLETED` (or `FAILED` with stored body), return stored `response_body` and status (`201` first success → `200` on replay).
4. **Conflict:** same key + different SHA-256 fingerprint of canonical `{fromWalletId,toWalletId,amount,currency}` → HTTP `409` (`ErrIdempotencyConflict`).
5. **Concurrent duplicates:** second request blocks on `FOR UPDATE` until the winner commits, then reads completed snapshot (no double debit).

## Concurrency

- **Isolation:** PostgreSQL default `READ COMMITTED` plus explicit row locks (not global `SERIALIZABLE`).
- **Wallet locks:** `SELECT … FOR UPDATE` on both wallets involved in the transfer.
- **Lock ordering:** wallet IDs sorted lexicographically before locking to avoid deadlocks on cross transfers (A→B and B→A).
- **Same-wallet debits:** serialized by source wallet lock; combined debits exceeding balance yield one `PROCESSED` and one `FAILED` (`409` response body) with unchanged balance after the failed attempt.

## Failure handling

| Case | HTTP | Transfer row | Ledger | Balances |
|------|------|--------------|--------|----------|
| Insufficient funds | 409 | `FAILED` + reason | none | unchanged |
| Missing wallet | 404 | `FAILED` + reason | none | unchanged |
| Validation (same wallet, bad currency) | 422 | none (pre-TX) | none | unchanged |
| Idempotency fingerprint mismatch | 409 | none | none | unchanged |
| Unexpected DB error | 500 | rolled back | none | unchanged |

Failed business outcomes still **commit** the `FAILED` transfer and **complete** idempotency so retries are stable.

## Deviations / choices

- **Idempotency key source:** accepts JSON `idempotencyKey` or `Idempotency-Key` header (header wins when set).
- **Failed transfer HTTP body:** returns full transfer JSON (status `FAILED`) rather than bare `{error:…}` so replays match success shape; status code carries semantics (`404`/`409`).
- **Transfer IDs:** prefixed `tr_` + UUID v4.
- **No auth** in v1 per DESIGN non-goals.
- **Integration tests** require running Postgres (`make docker-up && make migrate-up`); default `go test ./...` excludes them via `integration` build tag.

## AI usage (PR checklist)

1. Used AI to scaffold repository `FOR UPDATE` methods and service transaction flow from `docs/DESIGN.md`.
2. Used AI to generate integration test matrix (idempotency, insufficient funds, concurrent debits) and README curl examples.
3. Manually reviewed lock ordering, idempotency replay status codes, and commit/rollback boundaries before marking phases complete.
