package wallet

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("wallet not found")

type Wallet struct {
	UserID    string
	Balance   int64
	UpdatedAt time.Time
}
