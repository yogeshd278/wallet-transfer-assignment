package postgres

import (
	"context"
	"errors"
	"time"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/repository"

	"github.com/jackc/pgx/v5"
)

// TransferRepo implements repository.TransferRepository.
type TransferRepo struct{}

func scanTransfer(row pgx.Row) (*domain.Transfer, error) {
	var t domain.Transfer
	var currency string
	var status string
	err := row.Scan(
		&t.ID,
		&t.IdempotencyKey,
		&t.FromWalletID,
		&t.ToWalletID,
		&t.Amount,
		&currency,
		&status,
		&t.FailureReason,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.ProcessedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Currency = domain.Currency(currency)
	t.Status = domain.TransferStatus(status)
	return &t, nil
}

const transferSelectCols = `
	id, idempotency_key, from_wallet_id, to_wallet_id, amount, currency,
	status, failure_reason, created_at, updated_at, processed_at`

func (TransferRepo) Create(ctx context.Context, q repository.DBTX, t *domain.Transfer) error {
	const sql = `
		INSERT INTO transfers (
			id, idempotency_key, from_wallet_id, to_wallet_id, amount, currency,
			status, failure_reason, created_at, updated_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), $9)`
	_, err := q.Exec(ctx, sql,
		t.ID,
		t.IdempotencyKey,
		t.FromWalletID,
		t.ToWalletID,
		t.Amount.Int64(),
		t.Currency.String(),
		string(t.Status),
		t.FailureReason,
		t.ProcessedAt,
	)
	return err
}

func (TransferRepo) GetByID(ctx context.Context, q repository.DBTX, id string) (*domain.Transfer, error) {
	sql := `SELECT ` + transferSelectCols + ` FROM transfers WHERE id = $1`
	t, err := scanTransfer(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransferNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (TransferRepo) GetByIdempotencyKey(ctx context.Context, q repository.DBTX, key string) (*domain.Transfer, error) {
	sql := `SELECT ` + transferSelectCols + ` FROM transfers WHERE idempotency_key = $1`
	t, err := scanTransfer(q.QueryRow(ctx, sql, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransferNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (TransferRepo) UpdateStatus(
	ctx context.Context,
	q repository.DBTX,
	id string,
	status domain.TransferStatus,
	failureReason *string,
	processedAt *time.Time,
) error {
	const sql = `
		UPDATE transfers
		SET status = $2, failure_reason = $3, processed_at = $4, updated_at = NOW()
		WHERE id = $1`
	tag, err := q.Exec(ctx, sql, id, string(status), failureReason, processedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTransferNotFound
	}
	return nil
}
