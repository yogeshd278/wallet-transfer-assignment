package domain

import "time"

// Wallet is a balance cache; the ledger is the audit source of truth.
type Wallet struct {
	ID        string
	Currency  Currency
	Balance   Money
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Transfer represents a wallet-to-wallet movement.
type Transfer struct {
	ID             string
	IdempotencyKey string
	FromWalletID   string
	ToWalletID     string
	Amount         Money
	Currency       Currency
	Status         TransferStatus
	FailureReason  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ProcessedAt    *time.Time
}

// LedgerEntry is an immutable ledger line tied to a processed transfer.
type LedgerEntry struct {
	ID         int64
	TransferID string
	WalletID   string
	EntryType  LedgerEntryType
	Amount     Money
	Currency   Currency
	CreatedAt  time.Time
}

// IdempotencyRecord stores in-flight and completed idempotent operations.
type IdempotencyRecord struct {
	IdempotencyKey     string
	RequestFingerprint string
	Status             IdempotencyStatus
	TransferID         *string
	ResponseStatus     *int
	ResponseBody       []byte
	CreatedAt          time.Time
	ExpiresAt          *time.Time
}

// ValidateTransferRequest checks business rules before persisting a transfer.
func ValidateTransferRequest(fromWalletID, toWalletID string, amount Money, currency Currency) error {
	if fromWalletID == "" {
		return &ValidationError{Field: "fromWalletId", Message: "is required"}
	}
	if toWalletID == "" {
		return &ValidationError{Field: "toWalletId", Message: "is required"}
	}
	if fromWalletID == toWalletID {
		return ErrSameWallet
	}
	if !amount.IsPositive() {
		return &ValidationError{Field: "amount", Message: "must be greater than zero"}
	}
	if currency == "" {
		return &ValidationError{Field: "currency", Message: "is required"}
	}
	return nil
}
