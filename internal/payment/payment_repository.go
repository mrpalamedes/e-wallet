package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, session Session, ttl time.Duration) (Session, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO transactions
			(user_id, type, amount, from_account, to_account, status, description, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), now() + make_interval(secs => $8))
		 RETURNING id, status, created_at, expires_at`, session.UserID, recordType, session.Amount, session.UserID, session.MerchantID, StatusPending, session.Description, ttl.Seconds(),
	).Scan(&session.ID, &session.Status, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return Session{}, fmt.Errorf("insert payment session: %w", err)
	}
	return session, nil
}

func (r *Repository) Confirm(ctx context.Context, userID, paymentID string) (ConfirmResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		amount  int64
		status  Status
		overdue bool
	)

	err = tx.QueryRow(ctx,
		`SELECT amount, status, expires_at <= now()
		 FROM transactions
		 WHERE id = $1 AND user_id = $2 AND type = $3
		 FOR UPDATE`, paymentID, userID, recordType,
	).Scan(&amount, &status, &overdue)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmResult{}, ErrNotFound
	}
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("lock payment session: %w", err)
	}

	switch status {
	case StatusPending:
	case StatusExpired:
		return ConfirmResult{}, ErrExpired
	default:
		return ConfirmResult{}, ErrNotPending
	}

	if overdue {
		if err := setStatus(ctx, tx, paymentID, StatusExpired); err != nil {
			return ConfirmResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ConfirmResult{}, fmt.Errorf("commit expiry: %w", err)
		}
		return ConfirmResult{}, ErrExpired
	}

	var balance int64
	err = tx.QueryRow(ctx,
		`SELECT balance
		 FROM wallets
		 WHERE user_id = $1
		 FOR UPDATE`, userID,
	).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmResult{}, ErrWalletNotFound
	}
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("lock wallet: %w", err)
	}
	if balance < amount {
		return ConfirmResult{}, ErrInsufficientFunds
	}

	result := ConfirmResult{TransactionID: paymentID}
	err = tx.QueryRow(ctx,
		`UPDATE wallets
		 SET balance = balance - $1, updated_at = now()
		 WHERE user_id = $2
		 RETURNING balance`, amount, userID,
	).Scan(&result.NewBalance)
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("debit wallet: %w", err)
	}

	if err := setStatus(ctx, tx, paymentID, StatusCompleted); err != nil {
		return ConfirmResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ConfirmResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return result, nil
}

func (r *Repository) ExpireOverdue(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE transactions
		 SET status = 'expired'
		 WHERE status = 'pending' AND type = 'payment' AND expires_at <= now()`,
	)
	if err != nil {
		return 0, fmt.Errorf("expire overdue payments: %w", err)
	}
	return tag.RowsAffected(), nil
}

func setStatus(ctx context.Context, tx pgx.Tx, paymentID string, status Status) error {
	_, err := tx.Exec(ctx,
		`UPDATE transactions SET status = $1 WHERE id = $2`,
		status, paymentID,
	)
	if err != nil {
		return fmt.Errorf("set payment status to %s: %w", status, err)
	}
	return nil
}
