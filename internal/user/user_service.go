package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const maxPasswordBytes = 72

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type repository interface {
	Create(ctx context.Context, user User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
}

type tokenGenerator interface {
	Generate(userID string) (string, error)
}

type Service struct {
	userRepository repository
	token          tokenGenerator
}

func NewService(userRepository repository, token tokenGenerator) *Service {
	return &Service{userRepository: userRepository, token: token}
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, error) {
	if len(input.Password) > maxPasswordBytes {
		return User{}, ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepository.Create(ctx, User{
		Name:         strings.TrimSpace(input.Name),
		Email:        normalizeEmail(input.Email),
		PasswordHash: string(hash),
	})
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *Service) Login(ctx context.Context, input LoginInput) (string, error) {
	if len(input.Password) > maxPasswordBytes {
		return "", ErrInvalidCredentials
	}

	user, err := s.userRepository.FindByEmail(ctx, normalizeEmail(input.Email))
	if errors.Is(err, ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(input.Password))
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("compare password: %w", err)
	}

	token, err := s.token.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return token, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
