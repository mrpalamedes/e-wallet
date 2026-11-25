package transaction

import (
	"context"
	"fmt"
	"strings"
)

const (
	defaultPage  = 1
	defaultLimit = 10
)

type repository interface {
	TopUp(ctx context.Context, userID string, amount int64) (TopUpResult, error)
	Transfer(ctx context.Context, fromID, toID string, amount int64) (string, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]Record, int, error)
}

type Service struct {
	transactionRepository repository
}

func NewService(transactionRepository repository) *Service {
	return &Service{transactionRepository: transactionRepository}
}

func (s *Service) TopUp(ctx context.Context, userID string, amount int64) (TopUpResult, error) {
	if amount < 1 {
		return TopUpResult{}, ErrInvalidAmount
	}

	result, err := s.transactionRepository.TopUp(ctx, userID, amount)
	if err != nil {
		return TopUpResult{}, fmt.Errorf("top up: %w", err)
	}
	return result, nil
}

func (s *Service) Transfer(ctx context.Context, fromID, toID string, amount int64) (string, error) {
	if amount < 1 {
		return "", ErrInvalidAmount
	}

	toID = strings.ToLower(strings.TrimSpace(toID))
	if toID == fromID {
		return "", ErrSelfTransfer
	}

	transactionID, err := s.transactionRepository.Transfer(ctx, fromID, toID, amount)
	if err != nil {
		return "", fmt.Errorf("transfer: %w", err)
	}
	return transactionID, nil
}

func (s *Service) History(ctx context.Context, userID string, page, limit int) (Page, error) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	}

	records, total, err := s.transactionRepository.ListByUser(ctx, userID, limit, (page-1)*limit)
	if err != nil {
		return Page{}, fmt.Errorf("list transactions: %w", err)
	}

	return Page{
		Records:     records,
		CurrentPage: page,
		TotalPages:  (total + limit - 1) / limit,
		TotalItems:  total,
	}, nil
}
