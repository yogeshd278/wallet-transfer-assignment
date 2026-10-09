package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/repository"
	"wallet_transfer_service/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTransferInput is the application-level transfer request.
type CreateTransferInput struct {
	IdempotencyKey string
	FromWalletID   string
	ToWalletID     string
	Amount         int64
	Currency       string
}

// CreateTransferResult holds the HTTP payload to return to the client.
type CreateTransferResult struct {
	Replay     bool
	StatusCode int
	Body       []byte
	TransferID string
}

// TransferService orchestrates wallet transfers inside database transactions.
type TransferService struct {
	pool *pgxpool.Pool
	w    repository.WalletRepository
	t    repository.TransferRepository
	l    repository.LedgerRepository
	i    repository.IdempotencyRepository
}

// NewTransferService wires repositories from a postgres Store.
func NewTransferService(store *postgres.Store) *TransferService {
	return &TransferService{
		pool: store.Pool,
		w:    store.Wallets,
		t:    store.Transfers,
		l:    store.Ledger,
		i:    store.Idempotency,
	}
}

// GetTransfer returns a transfer by id (read-only).
func (s *TransferService) GetTransfer(ctx context.Context, id string) (*domain.Transfer, error) {
	return s.t.GetByID(ctx, s.pool, id)
}

// CreateTransfer executes or replays an idempotent wallet transfer.
func (s *TransferService) CreateTransfer(ctx context.Context, in CreateTransferInput) (*CreateTransferResult, error) {
	currency, err := domain.ParseCurrency(in.Currency)
	if err != nil {
		return nil, err
	}
	amount, err := domain.NewTransferAmount(in.Amount)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateTransferRequest(in.FromWalletID, in.ToWalletID, amount, currency); err != nil {
		return nil, err
	}
	fingerprint, err := domain.RequestFingerprint(in.FromWalletID, in.ToWalletID, in.Amount, in.Currency)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	replay, err := s.claimOrReplayIdempotency(ctx, tx, in.IdempotencyKey, fingerprint)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		committed = true
		return replay, nil
	}

	transferID := "tr_" + uuid.NewString()
	now := time.Now().UTC()
	transfer := &domain.Transfer{
		ID:             transferID,
		IdempotencyKey: in.IdempotencyKey,
		FromWalletID:   in.FromWalletID,
		ToWalletID:     in.ToWalletID,
		Amount:         amount,
		Currency:       currency,
		Status:         domain.TransferStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.t.Create(ctx, tx, transfer); err != nil {
		return nil, err
	}

	fromWallet, toWallet, bizErr := s.lockAndLoadWallets(ctx, tx, in.FromWalletID, in.ToWalletID)
	if bizErr != nil {
		return s.commitFailed(ctx, tx, &committed, transfer, in.IdempotencyKey, fingerprint, bizErr)
	}
	if err := domain.CurrenciesMatch(fromWallet.Currency, currency); err != nil {
		return s.commitFailed(ctx, tx, &committed, transfer, in.IdempotencyKey, fingerprint, err)
	}
	if err := domain.CurrenciesMatch(toWallet.Currency, currency); err != nil {
		return s.commitFailed(ctx, tx, &committed, transfer, in.IdempotencyKey, fingerprint, err)
	}
	if fromWallet.Balance.Int64() < amount.Int64() {
		return s.commitFailed(ctx, tx, &committed, transfer, in.IdempotencyKey, fingerprint, domain.ErrInsufficientFunds)
	}

	newFromBalance := domain.Money(fromWallet.Balance.Int64() - amount.Int64())
	newToBalance := domain.Money(toWallet.Balance.Int64() + amount.Int64())
	if err := s.w.UpdateBalance(ctx, tx, fromWallet.ID, newFromBalance, fromWallet.Version); err != nil {
		return nil, err
	}
	if err := s.w.UpdateBalance(ctx, tx, toWallet.ID, newToBalance, toWallet.Version); err != nil {
		return nil, err
	}

	debit := &domain.LedgerEntry{
		TransferID: transferID,
		WalletID:   in.FromWalletID,
		EntryType:  domain.LedgerEntryDebit,
		Amount:     amount,
		Currency:   currency,
	}
	credit := &domain.LedgerEntry{
		TransferID: transferID,
		WalletID:   in.ToWalletID,
		EntryType:  domain.LedgerEntryCredit,
		Amount:     amount,
		Currency:   currency,
	}
	if err := s.l.Insert(ctx, tx, debit); err != nil {
		return nil, err
	}
	if err := s.l.Insert(ctx, tx, credit); err != nil {
		return nil, err
	}

	processedAt := time.Now().UTC()
	if err := s.t.UpdateStatus(ctx, tx, transferID, domain.TransferStatusProcessed, nil, &processedAt); err != nil {
		return nil, err
	}
	transfer.Status = domain.TransferStatusProcessed
	transfer.ProcessedAt = &processedAt

	body, statusCode, err := marshalTransferResponse(transfer)
	if err != nil {
		return nil, err
	}
	if err := s.completeIdempotency(ctx, tx, in.IdempotencyKey, fingerprint, &transferID, statusCode, body); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	committed = true

	slog.InfoContext(ctx, "transfer processed",
		slog.String("transfer_id", transferID),
		slog.String("idempotency_key", in.IdempotencyKey),
		slog.String("from_wallet_id", in.FromWalletID),
		slog.String("to_wallet_id", in.ToWalletID),
		slog.Int64("amount", in.Amount),
		slog.String("status", string(transfer.Status)),
	)

	return &CreateTransferResult{
		StatusCode: statusCode,
		Body:       body,
		TransferID: transferID,
	}, nil
}

func (s *TransferService) claimOrReplayIdempotency(
	ctx context.Context,
	tx pgx.Tx,
	key, fingerprint string,
) (*CreateTransferResult, error) {
	record := &domain.IdempotencyRecord{
		IdempotencyKey:     key,
		RequestFingerprint: fingerprint,
		Status:             domain.IdempotencyStatusInProgress,
	}
	err := s.i.Insert(ctx, tx, record)
	if err == nil {
		return nil, nil
	}
	if !isUniqueViolation(err) {
		return nil, err
	}

	existing, err := s.i.GetByKeyForUpdate(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	if existing.RequestFingerprint != fingerprint {
		return nil, domain.ErrIdempotencyConflict
	}
	if existing.Status == domain.IdempotencyStatusInProgress {
		return nil, fmt.Errorf("idempotency record still in progress after lock")
	}
	return replayFromRecord(existing)
}

func replayFromRecord(r *domain.IdempotencyRecord) (*CreateTransferResult, error) {
	if r.ResponseStatus == nil || r.ResponseBody == nil {
		return nil, fmt.Errorf("completed idempotency record missing response")
	}
	tid := ""
	if r.TransferID != nil {
		tid = *r.TransferID
	}
	status := *r.ResponseStatus
	if status == 201 {
		status = 200
	}
	return &CreateTransferResult{
		Replay:     true,
		StatusCode: status,
		Body:       append([]byte(nil), r.ResponseBody...),
		TransferID: tid,
	}, nil
}

func (s *TransferService) completeIdempotency(
	ctx context.Context,
	tx pgx.Tx,
	key, fingerprint string,
	transferID *string,
	httpStatus int,
	body []byte,
) error {
	st := httpStatus
	record := &domain.IdempotencyRecord{
		IdempotencyKey:     key,
		RequestFingerprint: fingerprint,
		Status:             domain.IdempotencyStatusCompleted,
		TransferID:         transferID,
		ResponseStatus:     &st,
		ResponseBody:       append([]byte(nil), body...),
	}
	return s.i.Update(ctx, tx, record)
}

func (s *TransferService) commitFailed(
	ctx context.Context,
	tx pgx.Tx,
	committed *bool,
	transfer *domain.Transfer,
	idempotencyKey, fingerprint string,
	bizErr error,
) (*CreateTransferResult, error) {
	result, err := s.finishFailed(ctx, tx, transfer, idempotencyKey, fingerprint, bizErr)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	*committed = true
	return result, nil
}

func (s *TransferService) finishFailed(
	ctx context.Context,
	tx pgx.Tx,
	transfer *domain.Transfer,
	idempotencyKey, fingerprint string,
	bizErr error,
) (*CreateTransferResult, error) {
	reason := bizErr.Error()
	var httpStatus int
	switch {
	case errors.Is(bizErr, domain.ErrInsufficientFunds):
		httpStatus = 409
	case errors.Is(bizErr, domain.ErrWalletNotFound):
		httpStatus = 404
	case errors.As(bizErr, new(*domain.ValidationError)):
		httpStatus = 422
	default:
		httpStatus = 409
	}

	if err := s.t.UpdateStatus(ctx, tx, transfer.ID, domain.TransferStatusFailed, &reason, nil); err != nil {
		return nil, err
	}
	transfer.Status = domain.TransferStatusFailed
	transfer.FailureReason = &reason

	body, _, err := marshalTransferResponse(transfer)
	if err != nil {
		return nil, err
	}
	if err := s.completeIdempotency(ctx, tx, idempotencyKey, fingerprint, &transfer.ID, httpStatus, body); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "transfer failed",
		slog.String("transfer_id", transfer.ID),
		slog.String("idempotency_key", idempotencyKey),
		slog.String("status", string(transfer.Status)),
		slog.String("reason", reason),
	)

	return &CreateTransferResult{
		StatusCode: httpStatus,
		Body:       body,
		TransferID: transfer.ID,
	}, nil
}

// lockWallets locks wallet rows in deterministic lexicographic order to avoid deadlocks.
func (s *TransferService) lockAndLoadWallets(
	ctx context.Context,
	tx pgx.Tx,
	fromID, toID string,
) (*domain.Wallet, *domain.Wallet, error) {
	firstID, secondID := fromID, toID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}
	first, err := s.w.GetForUpdate(ctx, tx, firstID)
	if err != nil {
		return nil, nil, err
	}
	second, err := s.w.GetForUpdate(ctx, tx, secondID)
	if err != nil {
		return nil, nil, err
	}
	if fromID == firstID {
		return first, second, nil
	}
	return second, first, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
