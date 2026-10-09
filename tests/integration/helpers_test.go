//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"wallet_transfer_service/internal/repository/postgres"
)

func testStore(t *testing.T) *postgres.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://wallet:wallet@localhost:5433/wallet_transfer?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := postgres.NewStore(ctx, url)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func resetDemoWallets(t *testing.T, store *postgres.Store) {
	t.Helper()
	ctx := context.Background()
	_, err := store.Pool.Exec(ctx, `
		UPDATE wallets SET balance = 100000, version = 0, updated_at = NOW() WHERE id = 'wallet_1';
		UPDATE wallets SET balance = 50000, version = 0, updated_at = NOW() WHERE id = 'wallet_2';
	`)
	if err != nil {
		t.Fatalf("reset wallets: %v", err)
	}
}

func countTransfersByKey(t *testing.T, store *postgres.Store, key string) int {
	t.Helper()
	var n int
	err := store.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM transfers WHERE idempotency_key = $1`, key).Scan(&n)
	if err != nil {
		t.Fatalf("count transfers: %v", err)
	}
	return n
}

func countLedgerForTransfer(t *testing.T, store *postgres.Store, transferID string) int {
	t.Helper()
	var n int
	err := store.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM ledger_entries WHERE transfer_id = $1`, transferID).Scan(&n)
	if err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	return n
}

func walletBalance(t *testing.T, store *postgres.Store, id string) int64 {
	t.Helper()
	var bal int64
	err := store.Pool.QueryRow(context.Background(),
		`SELECT balance FROM wallets WHERE id = $1`, id).Scan(&bal)
	if err != nil {
		t.Fatalf("wallet balance: %v", err)
	}
	return bal
}
