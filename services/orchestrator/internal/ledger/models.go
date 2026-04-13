package ledger

import "github.com/google/uuid"

type TransferRequest struct {
	FromAccountID uuid.UUID
	ToAccountID uuid.UUID
	Amount int64
	Currency string
	IdempotencyKey string
	ReferenceID  string
}