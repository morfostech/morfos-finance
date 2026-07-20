package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.ExpenseCategory, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, nome FROM expense_categories ORDER BY nome`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []domain.ExpenseCategory
	for rows.Next() {
		var c domain.ExpenseCategory
		if err := rows.Scan(&c.ID, &c.Nome); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *CategoryRepository) Create(ctx context.Context, nome string) (*domain.ExpenseCategory, error) {
	var c domain.ExpenseCategory
	err := r.pool.QueryRow(ctx,
		`INSERT INTO expense_categories (nome) VALUES ($1) RETURNING id, nome`, nome).
		Scan(&c.ID, &c.Nome)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: categoria já existe", domain.ErrValidation)
		}
		return nil, fmt.Errorf("create category: %w", err)
	}
	return &c, nil
}

// Delete removes a category. Fails with a conflict if it is still referenced by
// transactions (FK), so financial history is never orphaned.
func (r *CategoryRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1`, id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return fmt.Errorf("%w: categoria em uso por transações", domain.ErrConflict)
		}
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
