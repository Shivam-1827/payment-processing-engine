package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/bank"
	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/ledger"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type PaymentWorker struct {
	reader        *kafka.Reader
	ledgerService ledger.LedgerService
	bankClient    bank.Client
	logger        *slog.Logger
}

func NewPaymentWorker(brokers []string, topic string, groupID string, ls ledger.LedgerService, bc bank.Client, logger *slog.Logger) *PaymentWorker {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		Topic:       topic,
		StartOffset: kafka.FirstOffset, // Ensure we process from the beginning if joining fresh
		MaxBytes:    10e6,              // 10MB
	})

	return &PaymentWorker{
		reader:        r,
		ledgerService: ls,
		bankClient:    bc,
		logger:        logger,
	}
}

// ValidatedEvent matches the JSON output from the C++ Sequencer
type ValidatedEvent struct {
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	Payload        struct {
		FromAccountID string `json:"from_account"`
		ToAccountID   string `json:"to_account"`
		Amount        int64  `json:"amount"`
		Currency      string `json:"currency"`
	} `json:"payload"`
}

func (w *PaymentWorker) Run(ctx context.Context) {
	w.logger.Info("Payment Worker started, consuming validated events...")

	for {
		// 1. Fetch message (DO NOT COMMIT YET)
		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // Context cancelled, shutdown gracefully
			}
			w.logger.Error("failed to fetch kafka message", "error", err)
			continue
		}

		// 2. Process Message
		err = w.processEvent(ctx, msg.Value)
		
		if err != nil {
			// If processing fails due to a systemic issue (DB down), we DO NOT commit the offset.
			// The container will crash, reboot, and re-consume this exact message so no money is lost.
			w.logger.Error("fatal processing error, halting worker", "error", err)
			panic(err) 
		}

		// 3. Commit Offset Exactly-Once
		// We only reach here if the Bank was charged AND Postgres safely committed.
		if err := w.reader.CommitMessages(ctx, msg); err != nil {
			w.logger.Error("failed to commit kafka offset", "error", err)
		}
	}
}

func (w *PaymentWorker) processEvent(ctx context.Context, data []byte) error {
	var event ValidatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		w.logger.Error("failed to unmarshal validated event, skipping", "error", err)
		return nil // Return nil so we commit the offset and drop the poison pill
	}

	// If C++ rejected it (e.g., velocity limits), ignore and commit offset
	if event.Status == "REJECTED" {
		w.logger.Warn("dropping rejected event", "key", event.IdempotencyKey, "reason", event.Reason)
		return nil
	}

	w.logger.Info("processing validated payment", "key", event.IdempotencyKey)

	// --- 1. Call External Bank with Exponential Backoff ---
	bankCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := w.chargeBankWithRetries(bankCtx, event)
	if err != nil {
		if err == bank.ErrCardDeclined {
			w.logger.Warn("payment failed: card declined", "key", event.IdempotencyKey)
			return nil // A definitive failure. Do not retry, just drop/commit.
		}
		// If it's a systemic timeout, return the error to trigger the panic and prevent offset commit
		return err 
	}

	// --- 2. Execute Local ACID Ledger Transaction ---
	fromAccountID, err := uuid.Parse(event.Payload.FromAccountID)
	if err != nil {
		w.logger.Error("invalid from_account UUID", "error", err, "key", event.IdempotencyKey)
		return nil // Invalid UUID format, skip this message
	}

	toAccountID, err := uuid.Parse(event.Payload.ToAccountID)
	if err != nil {
		w.logger.Error("invalid to_account UUID", "error", err, "key", event.IdempotencyKey)
		return nil // Invalid UUID format, skip this message
	}

	transferReq := ledger.TransferRequest{
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		Amount:         event.Payload.Amount,
		Currency:       event.Payload.Currency,
		IdempotencyKey: event.IdempotencyKey, // Used for exact-once DB constraint
		ReferenceID:    "ext_bank_" + event.IdempotencyKey[:8], 
	}

	err = w.ledgerService.Transfer(ctx, transferReq)
	if err != nil {
		if err == ledger.ErrDuplicateRequest || err == ledger.ErrInsufficientFunds {
			w.logger.Warn("ledger rejected transfer", "error", err, "key", event.IdempotencyKey)
			return nil // Business logic rejection, safe to commit offset
		}
		// DB Connection failed. Panic and don't commit offset.
		return err 
	}

	w.logger.Info("payment fully settled in DB", "key", event.IdempotencyKey)
	return nil
}

// chargeBankWithRetries implements a 3-attempt exponential backoff
func (w *PaymentWorker) chargeBankWithRetries(ctx context.Context, event ValidatedEvent) error {
	maxRetries := 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := w.bankClient.Charge(ctx, event.Payload.Amount, event.Payload.Currency, event.IdempotencyKey)
		if err == nil {
			return nil // Success
		}
		if err == bank.ErrCardDeclined {
			return err // Do not retry a declined card
		}

		w.logger.Warn("bank charge failed, retrying...", "attempt", attempt, "error", err)
		if attempt == maxRetries {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	return nil
}

func (w *PaymentWorker) Close() error {
	return w.reader.Close()
}