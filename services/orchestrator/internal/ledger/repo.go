package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository{
	return &Repository{db: db}
}

func (r *Repository) Transfer(ctx context.Context, req TransferRequest) error {
	if req.Amount <= 0 {
		return ErrInvalidAmount
	}

	if req.FromAccountID == req.ToAccountID {
		return ErrSameAccount
	}

	for i:=0; i < 3; i++{
		err := r.transferTx(ctx, req)
		if err == nil { return nil}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40001" {
			time.Sleep(time.Millisecond * 50)
			continue
		}
		return err
	}
	return fmt.Errorf("transaction failed after retries")
}

func (r *Repository) transferTx(ctx context.Context, req TransferRequest) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Idempotency Check & Transaction Insert
	var txnID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_transactions (idempotency_key, reference_id, status, currency, amount)
		VALUES ($1, $2, 'SUCCESS', $3, $4)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`, req.IdempotencyKey, req.ReferenceID, req.Currency, req.Amount).Scan(&txnID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDuplicateRequest
		}
		return err
	}

	// 2. Deadlock Prevention: Always lock the smaller UUID first
	var firstLock, secondLock uuid.UUID
	if req.FromAccountID.String() < req.ToAccountID.String() {
		firstLock, secondLock = req.FromAccountID, req.ToAccountID
	} else {
		firstLock, secondLock = req.ToAccountID, req.FromAccountID
	}

	_, err = tx.Exec(ctx, "SELECT id FROM accounts WHERE id = ANY($1) FOR UPDATE", []uuid.UUID{firstLock, secondLock})
	if err != nil { return err }

	// 3. Balance & Currency Check
	var currentBalance int64
	var dbCurrency string
	err = tx.QueryRow(ctx, "SELECT balance, currency FROM accounts WHERE id = $1", req.FromAccountID).Scan(&currentBalance, &dbCurrency)
	if err != nil { return err }
	if dbCurrency != req.Currency { return ErrCurrencyMismatch }
	if currentBalance < req.Amount { return ErrInsufficientFunds }

	// 4. Update Balances
	_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1, version = version + 1 WHERE id = $2", req.Amount, req.FromAccountID)
	if err != nil { return err }
	_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1, version = version + 1 WHERE id = $2", req.Amount, req.ToAccountID)
	if err != nil { return err }

	// 5. Insert Double-Entry Ledger Rows
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries(transaction_id, account_id, amount)
		VALUES ($1, $2, $3), ($1, $4, $5)`, 
		txnID, req.FromAccountID, -req.Amount, req.ToAccountID, req.Amount)
	if err != nil { return err }

	// 6. OUTBOX PATTERN: Publish to Kafka Event Log securely
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ('payment', $1, 'PAYMENT_COMPLETED', jsonb_build_object('transaction_id', $1, 'amount', $2))`, 
		txnID, req.Amount)
	if err != nil { return err }

	return tx.Commit(ctx)
}