package domain

import (
	"errors"
	"fmt"
)

// ValidationError is a field-level validation failure for HTTP 422 mapping.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

var (
	ErrSameWallet              = errors.New("from and to wallet must differ")
	ErrInsufficientFunds       = errors.New("insufficient balance")
	ErrWalletNotFound          = errors.New("wallet not found")
	ErrTransferNotFound        = errors.New("transfer not found")
	ErrIdempotencyNotFound     = errors.New("idempotency record not found")
	ErrInvalidStatusTransition = errors.New("invalid transfer status transition")
	ErrIdempotencyConflict     = errors.New("idempotency key reused with different request")
)
