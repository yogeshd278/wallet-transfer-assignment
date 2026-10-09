package service

import (
	"encoding/json"
	"time"

	"wallet_transfer_service/internal/domain"
)

// TransferResponse is the JSON body for successful transfer operations.
type TransferResponse struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	FromWalletID   string     `json:"fromWalletId"`
	ToWalletID     string     `json:"toWalletId"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	IdempotencyKey string     `json:"idempotencyKey"`
	CreatedAt      time.Time  `json:"createdAt"`
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
	FailureReason  *string    `json:"failureReason,omitempty"`
}

// ErrorResponse is a minimal API error body.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MarshalTransferResponse encodes a transfer for HTTP responses.
func MarshalTransferResponse(t *domain.Transfer) ([]byte, int, error) {
	return marshalTransferResponse(t)
}

func marshalTransferResponse(t *domain.Transfer) ([]byte, int, error) {
	body := TransferResponse{
		ID:             t.ID,
		Status:         string(t.Status),
		FromWalletID:   t.FromWalletID,
		ToWalletID:     t.ToWalletID,
		Amount:         t.Amount.Int64(),
		Currency:       t.Currency.String(),
		IdempotencyKey: t.IdempotencyKey,
		CreatedAt:      t.CreatedAt,
		ProcessedAt:    t.ProcessedAt,
		FailureReason:  t.FailureReason,
	}
	status := 201
	if t.Status == domain.TransferStatusFailed {
		status = 409
	}
	b, err := json.Marshal(body)
	return b, status, err
}

func marshalErrorResponse(httpStatus int, message string) ([]byte, error) {
	return json.Marshal(ErrorResponse{Error: message})
}
