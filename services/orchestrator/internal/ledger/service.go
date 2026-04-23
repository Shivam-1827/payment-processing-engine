package ledger

import "context"

type LedgerService interface {
	Transfer(ctx context.Context, req TransferRequest) error
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest) error {
	return s.repo.Transfer(ctx, req)
}
