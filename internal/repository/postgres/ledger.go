package postgres

import (
	"context"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/repository"
)

// LedgerRepo implements repository.LedgerRepository.
type LedgerRepo struct{}

func (LedgerRepo) Insert(ctx context.Context, q repository.DBTX, e *domain.LedgerEntry) error {
	const sql = `
		INSERT INTO ledger_entries (transfer_id, wallet_id, entry_type, amount, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at`
	var entryType string = string(e.EntryType)
	return q.QueryRow(ctx, sql,
		e.TransferID,
		e.WalletID,
		entryType,
		e.Amount.Int64(),
		e.Currency.String(),
	).Scan(&e.ID, &e.CreatedAt)
}

func (LedgerRepo) ListByTransferID(ctx context.Context, q repository.DBTX, transferID string) ([]domain.LedgerEntry, error) {
	const sql = `
		SELECT id, transfer_id, wallet_id, entry_type, amount, currency, created_at
		FROM ledger_entries WHERE transfer_id = $1 ORDER BY id`
	rows, err := q.Query(ctx, sql, transferID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.LedgerEntry
	for rows.Next() {
		var e domain.LedgerEntry
		var entryType, currency string
		if err := rows.Scan(
			&e.ID, &e.TransferID, &e.WalletID, &entryType, &e.Amount, &currency, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.EntryType = domain.LedgerEntryType(entryType)
		e.Currency = domain.Currency(currency)
		out = append(out, e)
	}
	return out, rows.Err()
}
