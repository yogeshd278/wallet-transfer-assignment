package domain

// LedgerEntryType is DEBIT or CREDIT (magnitude is always positive in storage).
type LedgerEntryType string

const (
	LedgerEntryDebit  LedgerEntryType = "DEBIT"
	LedgerEntryCredit LedgerEntryType = "CREDIT"
)

// ValidLedgerEntryType reports whether t is DEBIT or CREDIT.
func ValidLedgerEntryType(t LedgerEntryType) bool {
	return t == LedgerEntryDebit || t == LedgerEntryCredit
}
