package domain

import "testing"

func TestLedgerPairBalancesToZero(t *testing.T) {
	amount := Money(10050)
	debit := LedgerEntry{EntryType: LedgerEntryDebit, Amount: amount}
	credit := LedgerEntry{EntryType: LedgerEntryCredit, Amount: amount}

	if debit.Amount != credit.Amount {
		t.Fatalf("debit amount %d != credit amount %d", debit.Amount, credit.Amount)
	}
	if !ValidLedgerEntryType(debit.EntryType) || !ValidLedgerEntryType(credit.EntryType) {
		t.Fatal("expected valid entry types")
	}
	signed := int64(debit.Amount)*-1 + int64(credit.Amount)
	if signed != 0 {
		t.Fatalf("signed sum should be 0, got %d", signed)
	}
}
