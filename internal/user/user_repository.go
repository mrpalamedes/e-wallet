package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user User) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		user.Name, user.Email, user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO wallets (user_id) VALUES ($1)`, user.ID); err != nil {
		return User{}, fmt.Errorf("insert wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit tx: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at
		 FROM users
		 WHERE email = $1`, email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("select user by email: %w", err)
	}
	return user, nil
}
