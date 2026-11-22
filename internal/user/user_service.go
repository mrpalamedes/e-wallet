package user

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const maxPasswordBytes = 72

type userRepository interface {
	Create(ctx context.Context, user User) (User, error)
}

type UserService struct {
	userRepository userRepository
}

func NewUserService(userRepository userRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) (User, error) {
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
