# Wallet Transfer Service (Coding Assignment)

Standalone service for **wallet-to-wallet transfers** with **exactly-once semantics** (client idempotency), a **double-entry ledger**, and **PostgreSQL** as the system of record. This folder is intentionally separate from production `wallet_service_be` (Razorpay, platform wallets) so the assignment can be reviewed in isolation.

## Status

**Phases 2–7 complete** — `POST /transfers`, `GET /transfers/:id`, idempotency, failure paths, concurrency tests, and docs. See [`docs/PLAN.md`](docs/PLAN.md) and [`docs/IMPLEMENTATION_NOTES.md`](docs/IMPLEMENTATION_NOTES.md).

## Documentation

| Document | Purpose |
|----------|---------|
| **[docs/PROJECT_GUIDE.md](docs/PROJECT_GUIDE.md)** | **Start here** — structure, features, how it works, scaling |
| [docs/PLAN.md](docs/PLAN.md) | Phased implementation plan, checklist, time estimates |
| [docs/DESIGN.md](docs/DESIGN.md) | API, schema, idempotency, concurrency, testing |
| [docs/TASKS.md](docs/TASKS.md) | Ordered task breakdown (requirement mapping) |
| [docs/IMPLEMENTATION_NOTES.md](docs/IMPLEMENTATION_NOTES.md) | PR notes: schema, idempotency, concurrency |
| [docs/PR_TEMPLATE.md](docs/PR_TEMPLATE.md) | PR description skeleton (incl. AI usage) |

## Stack

- **Go 1.22+**, **Gin**
- **PostgreSQL 15+** for runtime and integration tests
- **[golang-migrate](https://github.com/golang-migrate/migrate)** for versioned SQL migrations
- **stdlib `slog`** for structured logs; **pgx** repository SQL

## Prerequisites

- Go 1.22+
- Docker (for local Postgres)
- `migrate` CLI (`make migrate-install`)

## How to run locally

```bash
cd wallet_transfer_service

cp .env.example .env
make docker-up
make migrate-up
make run
```

Health:

```bash
curl -s http://localhost:8080/health
# {"status":"ok","db":"ok"}
```

### Example transfers

Successful transfer (demo wallets `wallet_1` → `wallet_2`, seeded with INR balances):

```bash
curl -s -X POST http://localhost:8080/transfers \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-transfer-001' \
  -d '{
    "fromWalletId": "wallet_1",
    "toWalletId": "wallet_2",
    "amount": 10050,
    "currency": "INR"
  }'
```

Replay (same key — no second debit):

```bash
curl -s -X POST http://localhost:8080/transfers \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-transfer-001' \
  -d '{
    "fromWalletId": "wallet_1",
    "toWalletId": "wallet_2",
    "amount": 10050,
    "currency": "INR"
  }'
```

Get transfer by id (use `id` from create response):

```bash
curl -s http://localhost:8080/transfers/tr_<uuid>
```

Insufficient funds (409, `FAILED` transfer, balances unchanged):

```bash
curl -s -X POST http://localhost:8080/transfers \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-insufficient-001' \
  -d '{
    "fromWalletId": "wallet_1",
    "toWalletId": "wallet_2",
    "amount": 999999999,
    "currency": "INR"
  }'
```

## Testing

```bash
# Unit + handler tests (no database required)
make test
# or: go test ./... -count=1

# Integration + concurrency (Postgres required)
make docker-up && make migrate-up
make test-integration
# or: go test ./tests/integration/... -tags=integration -race -count=1
```

`DATABASE_URL` defaults to `postgres://wallet:wallet@localhost:5433/wallet_transfer?sslmode=disable`.

## Concurrency & locking

- Isolation: **READ COMMITTED** (Postgres default) with **`SELECT … FOR UPDATE`** on both wallets.
- **Lock ordering:** wallet IDs are locked in lexicographic order to prevent deadlocks.
- Idempotent duplicates serialize on the idempotency row (`INSERT` + `FOR UPDATE` on conflict).

## Makefile targets

| Target | Description |
|--------|-------------|
| `make build` | `go build ./...` |
| `make run` | Start `cmd/server` |
| `make test` | Unit tests |
| `make test-integration` | Postgres integration tests (`-tags=integration -race`) |
| `make migrate-up` / `migrate-down` | SQL migrations |
| `make docker-up` / `docker-down` | Postgres compose |

## Repository layout

```
wallet_transfer_service/
├── cmd/server/
├── internal/{config,domain,handler,middleware,service,repository}
├── migrations/
├── tests/integration/     # build tag: integration
└── docs/
```

## Relation to PowerXchange monorepo

| Sibling | Relevance |
|---------|-----------|
| `wallet_service_be` | Platform wallet (float balances) — **not** used for this assignment |
| `order_service_be` | Settlement / idempotency patterns |
| `user_service_be` | Handler style reference |

This assignment uses **minor-unit balances** and a **formal double-entry ledger**.
