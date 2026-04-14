package ledger

import (
	"context"
	"log"
)

type LedgerService interface {
	Transfer(ctx context.Context, req TransferRequest) error
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service{
	return &Service{repo: repo}
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest) error {
	log.Printf("Initiating transfer of %d %s from %s to %s [IdempotencyKey: %s]", 
		req.Amount, req.Currency, req.FromAccountID, req.ToAccountID, req.IdempotencyKey)
	
	err := s.repo.Transfer(ctx, req)

	if err != nil {
		log.Printf("Transfer failed: %v", err);
		return err
	}

	log.Printf("Transfer successful [IdempotencyKey: %s]", req.IdempotencyKey)
	return nil
}