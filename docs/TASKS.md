# Task breakdown (ordered)

Mapping: **Red** = correctness/risk, **Blue** = structure/docs, **Green** = tests/CI. Adjust labels to match your assignment rubric if provided.

## 1. Project bootstrap (Blue)

| # | Task | Done |
|---|------|------|
| 1.1 | Create `go.mod`, `cmd/server/main.go` | [ ] |
| 1.2 | `docker-compose.yml` for Postgres | [ ] |
| 1.3 | `.env.example`, `.gitignore` | [ ] |
| 1.4 | `Makefile`: run, test, migrate | [ ] |
| 1.5 | `GET /health` | [ ] |

## 2. Schema & migrations (Red)

| # | Task | Done |
|---|------|------|
| 2.1 | Migration: `wallets` | [ ] |
| 2.2 | Migration: `transfers` + status CHECK + unique `idempotency_key` | [ ] |
| 2.3 | Migration: `ledger_entries` + uniqueness per transfer side | [ ] |
| 2.4 | Migration: `idempotency_records` | [ ] |
| 2.5 | Seed wallets for local demo (optional SQL) | [ ] |

## 3. Domain layer (Red / Blue)

| # | Task | Done |
|---|------|------|
| 3.1 | `Money` type (int64 minor units), currency validation | [ ] |
| 3.2 | `TransferStatus` enum + transitions | [ ] |
| 3.3 | Domain errors: `InsufficientFunds`, `WalletNotFound`, `IdempotencyConflict` | [ ] |
| 3.4 | Unit tests for validation rules | [ ] |

## 4. Repository layer (Red)

| # | Task | Done |
|---|------|------|
| 4.1 | `WalletRepository`: GetForUpdate, UpdateBalance | [ ] |
| 4.2 | `TransferRepository`: Insert, UpdateStatus, GetByID, GetByIdempotencyKey | [ ] |
| 4.3 | `LedgerRepository`: InsertDebitCreditPair | [ ] |
| 4.4 | `IdempotencyRepository`: Claim, Complete, Get | [ ] |
| 4.5 | All queries use transactions passed from service | [ ] |

## 5. Service layer — `CreateTransfer` (Red)

| # | Task | Done |
|---|------|------|
| 5.1 | Begin transaction | [ ] |
| 5.2 | Claim idempotency key (handle duplicate) | [ ] |
| 5.3 | Insert `PENDING` transfer | [ ] |
| 5.4 | Lock wallets, verify currency, check balance | [ ] |
| 5.5 | Update balances | [ ] |
| 5.6 | Insert ledger DEBIT + CREDIT | [ ] |
| 5.7 | Set transfer `PROCESSED`, complete idempotency record | [ ] |
| 5.8 | Commit; on business error mark `FAILED` or rollback per design | [ ] |

## 6. HTTP handler (Blue)

| # | Task | Done |
|---|------|------|
| 6.1 | `POST /transfers` — bind JSON, extract idempotency key | [ ] |
| 6.2 | Map service errors → HTTP status | [ ] |
| 6.3 | `GET /transfers/:id` | [ ] |
| 6.4 | Request logging + recovery middleware | [ ] |

## 7. Idempotency (Red)

| # | Task | Done |
|---|------|------|
| 7.1 | Request fingerprint (canonical JSON hash) | [ ] |
| 7.2 | Store response body + status on success | [ ] |
| 7.3 | Replay path returns stored response | [ ] |
| 7.4 | Conflict when fingerprint differs | [ ] |

## 8. Concurrency (Red)

| # | Task | Done |
|---|------|------|
| 8.1 | `FOR UPDATE` on wallet rows | [ ] |
| 8.2 | Deterministic lock order on two wallets | [ ] |
| 8.3 | Document isolation level in README/DESIGN | [ ] |

## 9. Tests (Green)

| # | Task | Done |
|---|------|------|
| 9.1 | Integration: successful transfer | [ ] |
| 9.2 | Integration: idempotency sequential retry | [ ] |
| 9.3 | Integration: parallel same idempotency key | [ ] |
| 9.4 | Integration: insufficient funds | [ ] |
| 9.5 | Integration: invalid / missing wallet | [ ] |
| 9.6 | Integration: concurrent debits same wallet | [ ] |
| 9.7 | Unit: ledger invariant (debit + credit = transfer amount) | [ ] |
| 9.8 | `go test -race` clean on integration package | [ ] |

## 10. Documentation & PR (Blue)

| # | Task | Done |
|---|------|------|
| 10.1 | README: run, migrate, curl examples | [ ] |
| 10.2 | Fill PR from `docs/PR_TEMPLATE.md` | [ ] |
| 10.3 | AI usage section (3 points) | [ ] |

## 11. Stretch goals (optional)

| # | Task | Done |
|---|------|------|
| 11.1 | `GET /wallets/:id` | [ ] |
| 11.2 | OpenAPI spec | [ ] |
| 11.3 | GitHub Actions CI + testcontainers | [ ] |
| 11.4 | Idempotency record TTL + cleanup cron | [ ] |
| 11.5 | Balance reconciliation job from ledger | [ ] |

## Suggested implementation order

1. Tasks **1 → 2 → 3 → 4 → 5 → 6** (vertical slice: one happy path)
2. **7** idempotency before declaring MVP
3. **8** concurrency before load-sensitive review
4. **9** expand tests in parallel with 5–8
5. **10** PR when 9.1–9.6 pass
