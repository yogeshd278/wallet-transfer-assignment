package domain

// TransferStatus is the lifecycle state of a transfer.
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "PENDING"
	TransferStatusProcessed TransferStatus = "PROCESSED"
	TransferStatusFailed    TransferStatus = "FAILED"
)

// ValidTransferStatus reports whether s is a known transfer status.
func ValidTransferStatus(s TransferStatus) bool {
	switch s {
	case TransferStatusPending, TransferStatusProcessed, TransferStatusFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo reports whether a transfer may move from current to next.
func (current TransferStatus) CanTransitionTo(next TransferStatus) bool {
	if current == next {
		return true
	}
	switch current {
	case TransferStatusPending:
		return next == TransferStatusProcessed || next == TransferStatusFailed
	case TransferStatusProcessed, TransferStatusFailed:
		return false
	default:
		return false
	}
}

// TransitionTransferStatus returns next or a domain error if the transition is invalid.
func TransitionTransferStatus(current, next TransferStatus) (TransferStatus, error) {
	if !ValidTransferStatus(next) {
		return current, &ValidationError{Field: "status", Message: "invalid transfer status"}
	}
	if !current.CanTransitionTo(next) {
		return current, ErrInvalidStatusTransition
	}
	return next, nil
}

// IdempotencyStatus tracks idempotent request processing.
type IdempotencyStatus string

const (
	IdempotencyStatusInProgress IdempotencyStatus = "IN_PROGRESS"
	IdempotencyStatusCompleted  IdempotencyStatus = "COMPLETED"
	IdempotencyStatusFailed     IdempotencyStatus = "FAILED"
)

// ValidIdempotencyStatus reports whether s is a known idempotency status.
func ValidIdempotencyStatus(s IdempotencyStatus) bool {
	switch s {
	case IdempotencyStatusInProgress, IdempotencyStatusCompleted, IdempotencyStatusFailed:
		return true
	default:
		return false
	}
}
