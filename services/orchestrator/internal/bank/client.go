package bank

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"time"
)

var (
	ErrBankTimeout     = errors.New("bank connection timeout")
	ErrBankUnavailable = errors.New("502 bad gateway: bank unavailable")
	ErrCardDeclined    = errors.New("card declined by issuer")
)

// Client defines how we talk to external payment rails (Visa, Mastercard, Stripe)
type Client interface {
	Charge(ctx context.Context, amount int64, currency string, referenceID string) error
}

type MockChaosBank struct {
	logger *slog.Logger
}

func NewMockChaosBank(logger *slog.Logger) *MockChaosBank {
	return &MockChaosBank{logger: logger}
}

func (b *MockChaosBank) Charge(ctx context.Context, amount int64, currency string, refID string) error {
	// Simulate network latency (50ms to 800ms)
	latency := time.Duration(rand.Intn(750)+50) * time.Millisecond
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(latency):
		// Simulate random chaos after latency
		chance := rand.Intn(100)

		if chance < 5 {
			b.logger.Warn("mock bank simulating 502 Bad Gateway", "ref", refID)
			return ErrBankUnavailable
		}
		if chance < 10 {
			b.logger.Warn("mock bank simulating network timeout", "ref", refID)
			return ErrBankTimeout
		}
		if chance < 12 {
			b.logger.Warn("mock bank simulating card decline", "ref", refID)
			return ErrCardDeclined
		}

		// 88% chance of success
		b.logger.Info("mock bank charge successful", "ref", refID, "amount", amount)
		return nil
	}
}