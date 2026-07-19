// Package repository provides data access over the pgx pool. Repositories return
// domain types and domain.ErrNotFound; callers never see pgx specifics.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const userColumns = `id, nome, email, senha_hash, role, must_change_password, ativo, created_at, updated_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Nome, &u.Email, &u.SenhaHash, &u.Role,
		&u.MustChangePassword, &u.Ativo, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (nome, email, senha_hash, role, must_change_password, ativo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+userColumns,
		u.Nome, u.Email, u.SenhaHash, u.Role, u.MustChangePassword, u.Ativo)
	created, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

// UpdatePassword sets a new hash and clears must_change_password in one shot.
func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET senha_hash = $1, must_change_password = FALSE WHERE id = $2`,
		hash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ResetPassword sets a new hash and forces a change on next login (admin action).
func (r *UserRepository) ResetPassword(ctx context.Context, id int64, hash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET senha_hash = $1, must_change_password = TRUE WHERE id = $2`,
		hash, id)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List returns all users ordered by name (admin-only listing).
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY nome`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
