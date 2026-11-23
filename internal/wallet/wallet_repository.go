package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository struct {
	pool *pgxpool.Pool
}

func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

func (r *WalletRepository) FindByUserID(ctx context.Context, userID string) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, balance, updated_at
		 FROM wallets
		 WHERE user_id = $1`,
		userID).Scan(&wallet.UserID, &wallet.Balance, &wallet.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("select wallet by user id: %w", err)
	}
	return wallet, nil
}
