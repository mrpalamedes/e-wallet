package user

import (
	"errors"
	"time"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrNotFound           = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordTooLong    = errors.New("password must not exceed 72 bytes")
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
