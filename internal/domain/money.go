package domain

import "fmt"

// Money is an amount in minor currency units (e.g. paise for INR).
type Money int64

// NewTransferAmount validates a positive transfer amount.
func NewTransferAmount(amount int64) (Money, error) {
	if amount <= 0 {
		return 0, &ValidationError{
			Field:   "amount",
			Message: "must be greater than zero",
		}
	}
	return Money(amount), nil
}

// Int64 returns the raw minor-unit value.
func (m Money) Int64() int64 {
	return int64(m)
}

// IsPositive reports whether the amount is strictly positive.
func (m Money) IsPositive() bool {
	return m > 0
}

// Currency is a 3-letter ISO 4217 code.
type Currency string

// ParseCurrency validates length and non-empty currency codes.
func ParseCurrency(s string) (Currency, error) {
	if len(s) != 3 {
		return "", &ValidationError{
			Field:   "currency",
			Message: "must be a 3-letter ISO code",
		}
	}
	return Currency(s), nil
}

func (c Currency) String() string {
	return string(c)
}

// CurrenciesMatch returns an error if wallet currencies differ from the transfer currency.
func CurrenciesMatch(walletCurrency Currency, other Currency) error {
	if walletCurrency != other {
		return &ValidationError{
			Field:   "currency",
			Message: fmt.Sprintf("wallet currency %s does not match %s", walletCurrency, other),
		}
	}
	return nil
}
