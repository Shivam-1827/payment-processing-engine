package ledger

import "errors"

var (
	ErrInvalidAmount = errors.New("invalid amount")
	ErrSameAccount   = errors.New("same account transfer not allowed")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrDuplicateRequest = errors.New("duplicate request idempotency key")
	ErrCurrencyMismatch = errors.New("currency mismatch")
)