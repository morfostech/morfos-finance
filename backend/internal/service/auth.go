// Package service holds business rules. Services depend on repository interfaces
// (not concrete pgx types) so they can be unit-tested with fakes.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/morfostech/morfos-finance/internal/auth"
	"github.com/morfostech/morfos-finance/internal/domain"
)

const minPasswordLen = 8

// UserRepo is the persistence contract the auth service needs.
type UserRepo interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	UpdatePassword(ctx context.Context, id int64, hash string) error
	ResetPassword(ctx context.Context, id int64, hash string) error
	List(ctx context.Context) ([]domain.User, error)
}

type AuthService struct {
	users  UserRepo
	tokens *auth.TokenManager
}

func NewAuthService(users UserRepo, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// LoginResult carries the issued token plus the user for the client.
type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      *domain.User
}

// Login validates credentials and issues a JWT. Returns ErrInvalidCredentials
// for both unknown email and wrong password (no user enumeration); inactive
// users are rejected distinctly since an admin controls that flag.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Hash a throwaway to keep timing similar to the found path.
			_, _ = auth.HashPassword(password)
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	ok, err := auth.VerifyPassword(password, u.SenhaHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}
	if !u.Ativo {
		return nil, domain.ErrUserInactive
	}

	token, exp, err := s.tokens.Issue(u)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, ExpiresAt: exp, User: u}, nil
}

// Me returns the current user by ID.
func (s *AuthService) Me(ctx context.Context, id int64) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// ChangePassword lets a user set their own password after confirming the
// current one. Clears must_change_password (covers the first-login flow).
func (s *AuthService) ChangePassword(ctx context.Context, id int64, current, next string) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	ok, err := auth.VerifyPassword(current, u.SenhaHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return domain.ErrWrongPassword
	}
	if err := validatePassword(next); err != nil {
		return err
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, id, hash)
}

// CreateUserInput is the admin's payload for provisioning a user.
type CreateUserInput struct {
	Nome         string
	Email        string
	SenhaInicial string
	Role         domain.Role
}

// CreateUser provisions a user with an initial password and forces a change on
// first login. Admin-only (enforced at the HTTP layer).
func (s *AuthService) CreateUser(ctx context.Context, in CreateUserInput) (*domain.User, error) {
	in.Nome = strings.TrimSpace(in.Nome)
	in.Email = normalizeEmail(in.Email)
	if in.Nome == "" {
		return nil, fmt.Errorf("%w: nome é obrigatório", domain.ErrValidation)
	}
	if !validEmail(in.Email) {
		return nil, fmt.Errorf("%w: e-mail inválido", domain.ErrValidation)
	}
	if !in.Role.Valid() {
		return nil, fmt.Errorf("%w: cargo inválido", domain.ErrValidation)
	}
	if err := validatePassword(in.SenhaInicial); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(in.SenhaInicial)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, &domain.User{
		Nome:               in.Nome,
		Email:              in.Email,
		SenhaHash:          hash,
		Role:               in.Role,
		MustChangePassword: true,
		Ativo:              true,
	})
}

// ResetPassword lets an admin set a new initial password for a user, forcing a
// change on their next login.
func (s *AuthService) ResetPassword(ctx context.Context, id int64, novaSenha string) error {
	if err := validatePassword(novaSenha); err != nil {
		return err
	}
	hash, err := auth.HashPassword(novaSenha)
	if err != nil {
		return err
	}
	return s.users.ResetPassword(ctx, id, hash)
}

// ListUsers returns all users (admin-only).
func (s *AuthService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

func validatePassword(p string) error {
	if len(p) < minPasswordLen {
		return fmt.Errorf("%w: senha deve ter ao menos %d caracteres", domain.ErrValidation, minPasswordLen)
	}
	return nil
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}

func validEmail(e string) bool {
	_, err := mail.ParseAddress(e)
	return err == nil
}
