package payment

import (
	"errors"
	"time"
)

const recordType = "payment"

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusExpired   Status = "expired"
)

var (
	ErrInvalidAmount     = errors.New("amount must be at least 1")
	ErrInvalidMerchant   = errors.New("merchantID is required")
	ErrNotFound          = errors.New("payment not found")
	ErrExpired           = errors.New("payment session has expired")
	ErrNotPending        = errors.New("payment is no longer pending")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrWalletNotFound    = errors.New("wallet not found")
)

type Session struct {
	ID          string
	UserID      string
	MerchantID  string
	Description string
	Amount      int64
	Status      Status
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type ConfirmResult struct {
	TransactionID string
	NewBalance    int64
}
