package transaction

import (
	"errors"
	"time"
)

type Type string

const (
	TypeTopUp       Type = "topup"
	TypeTransferIn  Type = "transfer_in"
	TypeTransferOut Type = "transfer_out"
	TypePayment     Type = "payment"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

var (
	ErrInvalidAmount     = errors.New("amount must be at least 1")
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrSelfTransfer      = errors.New("cannot transfer to yourself")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrRecipientNotFound = errors.New("recipient not found")
)

type Record struct {
	ID        string
	UserID    string
	Type      Type
	Amount    int64
	From      string
	To        string
	Status    Status
	CreatedAt time.Time
}

type TopUpResult struct {
	TransactionID string
	NewBalance    int64
}

type Page struct {
	Records     []Record
	CurrentPage int
	TotalPages  int
	TotalItems  int
}
