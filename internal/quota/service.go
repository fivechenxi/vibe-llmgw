package quota

import (
	"context"
	"errors"
)

var ErrQuotaExceeded = errors.New("quota exceeded")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Check returns an error if the user has no remaining quota for the model.
func (s *Service) Check(ctx context.Context, userID, modelID string) error {
	q, err := s.repo.Get(ctx, userID, modelID)
	if err != nil {
		return err
	}
	if q.Remaining() <= 0 {
		return ErrQuotaExceeded
	}
	return nil
}

// Deduct subtracts consumed tokens after a successful request.
func (s *Service) Deduct(ctx context.Context, userID, modelID string, tokens int) error {
	return s.repo.Deduct(ctx, userID, modelID, tokens)
}