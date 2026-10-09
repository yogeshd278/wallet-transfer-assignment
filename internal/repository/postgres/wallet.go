package postgres

import (
	"context"
	"errors"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/repository"

	"github.com/jackc/pgx/v5"
)

// WalletRepo implements repository.WalletRepository.
type WalletRepo struct{}

func scanWallet(row pgx.Row) (*domain.Wallet, error) {
	var w domain.Wallet
	var currency string
	err := row.Scan(
		&w.ID, &currency, &w.Balance, &w.Version, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	w.Currency = domain.Currency(currency)
	return &w, nil
}

func (WalletRepo) GetByID(ctx context.Context, q repository.DBTX, id string) (*domain.Wallet, error) {
	const sql = `
		SELECT id, currency, balance, version, created_at, updated_at
		FROM wallets WHERE id = $1`
	w, err := scanWallet(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (WalletRepo) GetForUpdate(ctx context.Context, q repository.DBTX, id string) (*domain.Wallet, error) {
	const sql = `
		SELECT id, currency, balance, version, created_at, updated_at
		FROM wallets WHERE id = $1 FOR UPDATE`
	w, err := scanWallet(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (WalletRepo) Create(ctx context.Context, q repository.DBTX, w *domain.Wallet) error {
	const sql = `
		INSERT INTO wallets (id, currency, balance, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())`
	_, err := q.Exec(ctx, sql, w.ID, w.Currency.String(), w.Balance.Int64(), w.Version)
	return err
}

func (WalletRepo) UpdateBalance(ctx context.Context, q repository.DBTX, id string, newBalance domain.Money, expectedVersion int64) error {
	const sql = `
		UPDATE wallets
		SET balance = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $3`
	tag, err := q.Exec(ctx, sql, id, newBalance.Int64(), expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrWalletNotFound
	}
	return nil
}
