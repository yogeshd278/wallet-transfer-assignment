package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// transferRequestFingerprint is the canonical shape hashed for idempotency conflict detection.
type transferRequestFingerprint struct {
	FromWalletID string `json:"fromWalletId"`
	ToWalletID   string `json:"toWalletId"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

// RequestFingerprint returns a SHA-256 hex digest of the canonical transfer request fields.
func RequestFingerprint(fromWalletID, toWalletID string, amount int64, currency string) (string, error) {
	payload := transferRequestFingerprint{
		FromWalletID: fromWalletID,
		ToWalletID:   toWalletID,
		Amount:       amount,
		Currency:     currency,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal fingerprint: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
