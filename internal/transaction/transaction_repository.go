package transaction

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) TopUp(ctx context.Context, userID string, amount int64) (TopUpResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return TopUpResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var res TopUpResult

	err = tx.QueryRow(ctx,
		`UPDATE wallets
			 SET balance = balance + $1, updated_at = now()
			 WHERE user_id = $2
			 RETURNING balance`,
		amount, userID).Scan(&res.NewBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopUpResult{}, ErrWalletNotFound
	}
	if err != nil {
		return TopUpResult{}, fmt.Errorf("update wallet balance: %w", err)
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (user_id, type, amount, to_account, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		userID, TypeTopUp, amount, userID, StatusCompleted,
	).Scan(&res.TransactionID)
	if err != nil {
		return TopUpResult{}, fmt.Errorf("insert topup transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return TopUpResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return res, nil
}

func (r *TransactionRepository) Transfer(ctx context.Context, fromID, toID string, amount int64) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	balances, err := lockWallet(ctx, tx, fromID, toID)
	if err != nil {
		return "", err
	}

	fromBalance, ok := balances[fromID]
	if !ok {
		return "", ErrWalletNotFound
	}
	if _, ok := balances[toID]; !ok {
		return "", ErrRecipientNotFound
	}
	if fromBalance < amount {
		return "", ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx,
		`UPDATE wallets
		 SET balance = balance - $1, updated_at = now()
		 WHERE user_id = $2`, amount, fromID,
	)
	if err != nil {
		return "", fmt.Errorf("debit sender wallet: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE wallets
		 SET balance = balance + $1, updated_at = now()
		 WHERE user_id = $2`, amount, fromID,
	)
	if err != nil {
		return "", fmt.Errorf("credit recipient wallet: %w", err)
	}

	var transactionID string
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (user_id, type, amount, from_account, to_account, status)
		 VALUE ($1, $2, $3, $4, $5, $6)
		 RETURNING id`, fromID, TypeTransferOut, amount, fromID, toID, StatusCompleted,
	).Scan(&transactionID)
	if err != nil {
		return "", fmt.Errorf("insert transfer_out transaction: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (user_id, type, amount, from_account, to_account, status)
		 VALUE ($1, $2, $3, $4, $5, $6)`, toID, TypeTransferIn, amount, fromID, toID, StatusCompleted,
	)
	if err != nil {
		return "", fmt.Errorf("insert transfer_in transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return transactionID, nil
}

func lockWallet(ctx context.Context, tx pgx.Tx, userIDs ...string) (map[string]int64, error) {
	ordered := slices.Clone(userIDs)
	slices.Sort(ordered)

	balances := make(map[string]int64, len(ordered))
	for _, id := range ordered {
		var balance int64
		err := tx.QueryRow(ctx,
			`SELECT balance 
			 FROM wallets 
			 WHERE user_id = $1 
			 FOR UPDATE`, id).Scan(&balance)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("lock wallet %s: %w", id, err)
		}
		balances[id] = balance
	}
	return balances, nil
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Record, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM transactions WHERE user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, amount,
		        COALESCE(from_account, ''), COALESCE(to_account, ''),
		        status, created_at
		 FROM transactions
		 WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("select transactions: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		err := rows.Scan(&rec.ID, &rec.UserID, &rec.Type, &rec.Amount,
			&rec.From, &rec.To, &rec.Status, &rec.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate transactions: %w", err)
	}

	return records, total, nil
}
