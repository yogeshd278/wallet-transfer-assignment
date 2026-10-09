//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/service"
)

func TestSuccessfulTransfer(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)

	key := fmt.Sprintf("idem-success-%s", t.Name())
	res, err := svc.CreateTransfer(context.Background(), service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         10050,
		Currency:       "INR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("status %d body %s", res.StatusCode, res.Body)
	}
	if countLedgerForTransfer(t, store, res.TransferID) != 2 {
		t.Fatal("expected two ledger rows")
	}
	if walletBalance(t, store, "wallet_1") != 100000-10050 {
		t.Fatalf("unexpected wallet_1 balance")
	}
	if walletBalance(t, store, "wallet_2") != 50000+10050 {
		t.Fatalf("unexpected wallet_2 balance")
	}
}

func TestIdempotencySequentialReplay(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)
	key := fmt.Sprintf("idem-seq-%s", t.Name())
	in := service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         500,
		Currency:       "INR",
	}
	first, err := svc.CreateTransfer(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateTransfer(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replay {
		t.Fatal("expected replay")
	}
	if second.StatusCode != 200 {
		t.Fatalf("replay status %d", second.StatusCode)
	}
	if string(first.Body) != string(second.Body) {
		t.Fatalf("bodies differ: %s vs %s", first.Body, second.Body)
	}
	if countTransfersByKey(t, store, key) != 1 {
		t.Fatal("expected one transfer row")
	}
}

func TestIdempotencyParallelSameKey(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)
	key := fmt.Sprintf("idem-parallel-%s", t.Name())
	in := service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         250,
		Currency:       "INR",
	}

	const workers = 25
	var wg sync.WaitGroup
	bodies := make([][]byte, workers)
	statuses := make([]int, workers)
	errs := make([]error, workers)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()
			res, err := svc.CreateTransfer(context.Background(), in)
			errs[idx] = err
			if res != nil {
				bodies[idx] = append([]byte(nil), res.Body...)
				statuses[idx] = res.StatusCode
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d: %v", i, err)
		}
	}
	if countTransfersByKey(t, store, key) != 1 {
		t.Fatalf("expected one transfer, got %d", countTransfersByKey(t, store, key))
	}
	first := string(bodies[0])
	for i := 1; i < workers; i++ {
		if string(bodies[i]) != first {
			t.Fatalf("response mismatch at %d", i)
		}
	}
}

func TestInsufficientFunds(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)
	key := fmt.Sprintf("idem-insufficient-%s", t.Name())
	res, err := svc.CreateTransfer(context.Background(), service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_1",
		ToWalletID:     "wallet_2",
		Amount:         200000,
		Currency:       "INR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 409 {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
	if walletBalance(t, store, "wallet_1") != 100000 {
		t.Fatal("balance should be unchanged")
	}
	var payload service.TransferResponse
	if err := json.Unmarshal(res.Body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != string(domain.TransferStatusFailed) {
		t.Fatalf("status %s", payload.Status)
	}
	if countLedgerForTransfer(t, store, res.TransferID) != 0 {
		t.Fatal("failed transfer must not create ledger rows")
	}
}

func TestInvalidWallet(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)
	key := fmt.Sprintf("idem-missing-wallet-%s", t.Name())
	res, err := svc.CreateTransfer(context.Background(), service.CreateTransferInput{
		IdempotencyKey: key,
		FromWalletID:   "wallet_missing",
		ToWalletID:     "wallet_2",
		Amount:         100,
		Currency:       "INR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
	if walletBalance(t, store, "wallet_2") != 50000 {
		t.Fatal("destination balance unchanged")
	}
}

func TestConcurrentDebitsSameWallet(t *testing.T) {
	store := testStore(t)
	resetDemoWallets(t, store)
	svc := service.NewTransferService(store)

	type outcome struct {
		status int
	}
	outcomes := make([]outcome, 2)
	var wg sync.WaitGroup
	var okCount int32
	wg.Add(2)
	requests := []service.CreateTransferInput{
		{
			IdempotencyKey: fmt.Sprintf("concurrent-a-%s", t.Name()),
			FromWalletID:   "wallet_1",
			ToWalletID:     "wallet_2",
			Amount:         70000,
			Currency:       "INR",
		},
		{
			IdempotencyKey: fmt.Sprintf("concurrent-b-%s", t.Name()),
			FromWalletID:   "wallet_1",
			ToWalletID:     "wallet_2",
			Amount:         70000,
			Currency:       "INR",
		},
	}
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			res, err := svc.CreateTransfer(context.Background(), requests[idx])
			if err != nil {
				t.Errorf("transfer %d: %v", idx, err)
				return
			}
			outcomes[idx] = outcome{status: res.StatusCode}
			if res.StatusCode == 201 {
				atomic.AddInt32(&okCount, 1)
			}
		}(i)
	}
	wg.Wait()

	if atomic.LoadInt32(&okCount) != 1 {
		t.Fatalf("expected exactly one success, successes=%d statuses=%v", okCount, outcomes)
	}
	if walletBalance(t, store, "wallet_1") != 30000 {
		t.Fatalf("wallet_1 balance expected 30000, got %d", walletBalance(t, store, "wallet_1"))
	}
}
