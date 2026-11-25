package payment

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type repository interface {
	Create(ctx context.Context, session Session, ttl time.Duration) (Session, error)
	Confirm(ctx context.Context, userID, paymentID string) (ConfirmResult, error)
}

type Service struct {
	repo repository
	ttl  time.Duration
}

func NewService(repo repository, ttl time.Duration) *Service {
	return &Service{repo: repo, ttl: ttl}
}

type CreateInput struct {
	MerchantID  string
	Amount      int64
	Description string
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (Session, error) {
	if in.Amount < 1 {
		return Session{}, ErrInvalidAmount
	}
	merchantID := strings.TrimSpace(in.MerchantID)
	if merchantID == "" {
		return Session{}, ErrInvalidMerchant
	}

	sess, err := s.repo.Create(ctx, Session{
		UserID:      userID,
		MerchantID:  merchantID,
		Amount:      in.Amount,
		Description: strings.TrimSpace(in.Description),
	}, s.ttl)
	if err != nil {
		return Session{}, fmt.Errorf("create payment session: %w", err)
	}
	return sess, nil
}

func (s *Service) Confirm(ctx context.Context, userID, paymentID string) (ConfirmResult, error) {
	res, err := s.repo.Confirm(ctx, userID, strings.ToLower(paymentID))
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("confirm payment: %w", err)
	}
	return res, nil
}
