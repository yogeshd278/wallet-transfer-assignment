package postgres

import (
	"context"
	"errors"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/repository"

	"github.com/jackc/pgx/v5"
)

// IdempotencyRepo implements repository.IdempotencyRepository.
type IdempotencyRepo struct{}

func scanIdempotency(row pgx.Row) (*domain.IdempotencyRecord, error) {
	var r domain.IdempotencyRecord
	var status string
	err := row.Scan(
		&r.IdempotencyKey,
		&r.RequestFingerprint,
		&status,
		&r.TransferID,
		&r.ResponseStatus,
		&r.ResponseBody,
		&r.CreatedAt,
		&r.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	r.Status = domain.IdempotencyStatus(status)
	return &r, nil
}

const idempotencySelectCols = `
	idempotency_key, request_fingerprint, status, transfer_id,
	response_status, response_body, created_at, expires_at`

func (IdempotencyRepo) GetByKey(ctx context.Context, q repository.DBTX, key string) (*domain.IdempotencyRecord, error) {
	sql := `SELECT ` + idempotencySelectCols + ` FROM idempotency_records WHERE idempotency_key = $1`
	r, err := scanIdempotency(q.QueryRow(ctx, sql, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrIdempotencyNotFound
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (IdempotencyRepo) GetByKeyForUpdate(ctx context.Context, q repository.DBTX, key string) (*domain.IdempotencyRecord, error) {
	sql := `SELECT ` + idempotencySelectCols + ` FROM idempotency_records WHERE idempotency_key = $1 FOR UPDATE`
	r, err := scanIdempotency(q.QueryRow(ctx, sql, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrIdempotencyNotFound
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (IdempotencyRepo) Insert(ctx context.Context, q repository.DBTX, r *domain.IdempotencyRecord) error {
	const sql = `
		INSERT INTO idempotency_records (
			idempotency_key, request_fingerprint, status, transfer_id,
			response_status, response_body, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7)`
	_, err := q.Exec(ctx, sql,
		r.IdempotencyKey,
		r.RequestFingerprint,
		string(r.Status),
		r.TransferID,
		r.ResponseStatus,
		r.ResponseBody,
		r.ExpiresAt,
	)
	return err
}

func (IdempotencyRepo) Update(ctx context.Context, q repository.DBTX, r *domain.IdempotencyRecord) error {
	const sql = `
		UPDATE idempotency_records
		SET request_fingerprint = $2, status = $3, transfer_id = $4,
		    response_status = $5, response_body = $6, expires_at = $7
		WHERE idempotency_key = $1`
	tag, err := q.Exec(ctx, sql,
		r.IdempotencyKey,
		r.RequestFingerprint,
		string(r.Status),
		r.TransferID,
		r.ResponseStatus,
		r.ResponseBody,
		r.ExpiresAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrIdempotencyNotFound
	}
	return nil
}
