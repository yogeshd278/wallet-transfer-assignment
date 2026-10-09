package repository

import (
	"context"
	"time"

	"wallet_transfer_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is implemented by pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// WalletRepository persists and loads wallets.
type WalletRepository interface {
	GetByID(ctx context.Context, q DBTX, id string) (*domain.Wallet, error)
	GetForUpdate(ctx context.Context, q DBTX, id string) (*domain.Wallet, error)
	Create(ctx context.Context, q DBTX, w *domain.Wallet) error
	UpdateBalance(ctx context.Context, q DBTX, id string, newBalance domain.Money, expectedVersion int64) error
}

// TransferRepository persists transfers.
type TransferRepository interface {
	Create(ctx context.Context, q DBTX, t *domain.Transfer) error
	GetByID(ctx context.Context, q DBTX, id string) (*domain.Transfer, error)
	GetByIdempotencyKey(ctx context.Context, q DBTX, key string) (*domain.Transfer, error)
	UpdateStatus(ctx context.Context, q DBTX, id string, status domain.TransferStatus, failureReason *string, processedAt *time.Time) error
}

// LedgerRepository appends ledger lines.
type LedgerRepository interface {
	Insert(ctx context.Context, q DBTX, e *domain.LedgerEntry) error
	ListByTransferID(ctx context.Context, q DBTX, transferID string) ([]domain.LedgerEntry, error)
}

// IdempotencyRepository tracks idempotent requests.
type IdempotencyRepository interface {
	GetByKey(ctx context.Context, q DBTX, key string) (*domain.IdempotencyRecord, error)
	GetByKeyForUpdate(ctx context.Context, q DBTX, key string) (*domain.IdempotencyRecord, error)
	Insert(ctx context.Context, q DBTX, r *domain.IdempotencyRecord) error
	Update(ctx context.Context, q DBTX, r *domain.IdempotencyRecord) error
}
