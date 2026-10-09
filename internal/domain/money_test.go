package domain

import (
	"errors"
	"testing"
)

func TestNewTransferAmount(t *testing.T) {
	m, err := NewTransferAmount(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != Money(100) {
		t.Fatalf("expected 100, got %d", m)
	}

	_, err = NewTransferAmount(0)
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "amount" {
		t.Fatalf("expected validation error on amount, got %v", err)
	}
}

func TestParseCurrency(t *testing.T) {
	c, err := ParseCurrency("INR")
	if err != nil || c != Currency("INR") {
		t.Fatalf("ParseCurrency INR: %v %q", err, c)
	}
	if _, err := ParseCurrency("IN"); err == nil {
		t.Fatal("expected error for short currency")
	}
}

func TestValidateTransferRequest(t *testing.T) {
	amount, _ := NewTransferAmount(50)
	currency, _ := ParseCurrency("INR")

	if err := ValidateTransferRequest("a", "b", amount, currency); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	if !errors.Is(ValidateTransferRequest("a", "a", amount, currency), ErrSameWallet) {
		t.Fatal("expected ErrSameWallet")
	}
}

func TestTransferStatusTransitions(t *testing.T) {
	if !TransferStatusPending.CanTransitionTo(TransferStatusProcessed) {
		t.Fatal("pending -> processed allowed")
	}
	if TransferStatusProcessed.CanTransitionTo(TransferStatusPending) {
		t.Fatal("processed -> pending not allowed")
	}
	next, err := TransitionTransferStatus(TransferStatusPending, TransferStatusProcessed)
	if err != nil || next != TransferStatusProcessed {
		t.Fatalf("transition: %v %q", err, next)
	}
	_, err = TransitionTransferStatus(TransferStatusProcessed, TransferStatusFailed)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}
