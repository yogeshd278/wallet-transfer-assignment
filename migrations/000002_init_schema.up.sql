-- Phase 1: core schema for wallets, transfers, ledger, idempotency.

CREATE TABLE wallets (
    id          TEXT PRIMARY KEY,
    currency    CHAR(3) NOT NULL,
    balance     BIGINT NOT NULL DEFAULT 0,
    version     BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT wallets_balance_non_negative CHECK (balance >= 0)
);

CREATE TABLE transfers (
    id               TEXT PRIMARY KEY,
    idempotency_key  TEXT NOT NULL,
    from_wallet_id   TEXT NOT NULL REFERENCES wallets (id),
    to_wallet_id     TEXT NOT NULL REFERENCES wallets (id),
    amount           BIGINT NOT NULL,
    currency         CHAR(3) NOT NULL,
    status           TEXT NOT NULL,
    failure_reason   TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at     TIMESTAMPTZ,
    CONSTRAINT transfers_status_check CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    CONSTRAINT transfers_amount_positive CHECK (amount > 0),
    CONSTRAINT transfers_distinct_wallets CHECK (from_wallet_id <> to_wallet_id),
    CONSTRAINT transfers_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX transfers_from_wallet_created_at_idx
    ON transfers (from_wallet_id, created_at DESC);

CREATE TABLE ledger_entries (
    id           BIGSERIAL PRIMARY KEY,
    transfer_id  TEXT NOT NULL REFERENCES transfers (id),
    wallet_id    TEXT NOT NULL REFERENCES wallets (id),
    entry_type   TEXT NOT NULL,
    amount       BIGINT NOT NULL,
    currency     CHAR(3) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ledger_entries_type_check CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ledger_entries_amount_positive CHECK (amount > 0),
    CONSTRAINT ledger_entries_transfer_type_wallet_unique
        UNIQUE (transfer_id, entry_type, wallet_id)
);

CREATE INDEX ledger_entries_transfer_id_idx ON ledger_entries (transfer_id);
CREATE INDEX ledger_entries_wallet_id_idx ON ledger_entries (wallet_id);

CREATE TABLE idempotency_records (
    idempotency_key      TEXT PRIMARY KEY,
    request_fingerprint  TEXT NOT NULL,
    status               TEXT NOT NULL,
    transfer_id          TEXT REFERENCES transfers (id),
    response_status      INT,
    response_body        JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at           TIMESTAMPTZ,
    CONSTRAINT idempotency_records_status_check
        CHECK (status IN ('IN_PROGRESS', 'COMPLETED', 'FAILED'))
);
