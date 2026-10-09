package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store bundles Postgres repository implementations and the connection pool.
type Store struct {
	Pool *pgxpool.Pool

	Wallets     WalletRepo
	Transfers   TransferRepo
	Ledger      LedgerRepo
	Idempotency IdempotencyRepo
}

// NewStore opens a pool and returns repositories sharing it.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{
		Pool:        pool,
		Wallets:     WalletRepo{},
		Transfers:   TransferRepo{},
		Ledger:      LedgerRepo{},
		Idempotency: IdempotencyRepo{},
	}, nil
}

// Close releases pool connections.
func (s *Store) Close() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
