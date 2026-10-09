//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/service"
)

func TestIdempotencyFingerprintConflict(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)
	key := fmt.Sprintf("idem-conflict-%s", t.Name())

	_, err := svc.CreateTransfer(context.Background(), service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         100,
		Currency:       "INR",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateTransfer(context.Background(), service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         200,
		Currency:       "INR",
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}
