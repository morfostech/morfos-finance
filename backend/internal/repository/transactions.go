package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

const txColumns = `id, tipo, valor::text, data, project_id, user_id, origem,
	category_id, descricao, installment_id, created_by, created_at, updated_at`

func scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var (
		t     domain.Transaction
		valor string
		data  time.Time
	)
	err := row.Scan(&t.ID, &t.Tipo, &valor, &data, &t.ProjectID, &t.UserID,
		&t.Origem, &t.CategoryID, &t.Descricao, &t.InstallmentID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	m, err := domain.ParseNumeric(valor)
	if err != nil {
		return nil, err
	}
	t.Valor = m
	t.Data = domain.NewDate(data)
	return &t, nil
}

func (r *TransactionRepository) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO transactions
			(tipo, valor, data, project_id, user_id, origem, category_id, descricao, created_by)
		VALUES ($1, $2::numeric, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+txColumns,
		t.Tipo, t.Valor.Numeric(), t.Data.Time, t.ProjectID, t.UserID,
		t.Origem, t.CategoryID, t.Descricao, t.CreatedBy)
	created, err := scanTransaction(row)
	if err != nil {
		return nil, mapTxFKError(err)
	}
	return created, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (*domain.Transaction, error) {
	return scanTransaction(r.pool.QueryRow(ctx,
		`SELECT `+txColumns+` FROM transactions WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *TransactionRepository) Update(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE transactions SET
			tipo = $2, valor = $3::numeric, data = $4, project_id = $5, user_id = $6,
			origem = $7, category_id = $8, descricao = $9
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+txColumns,
		t.ID, t.Tipo, t.Valor.Numeric(), t.Data.Time, t.ProjectID, t.UserID,
		t.Origem, t.CategoryID, t.Descricao)
	updated, err := scanTransaction(row)
	if err != nil {
		return nil, mapTxFKError(err)
	}
	return updated, nil
}

// SoftDelete marks a transaction deleted without removing the row.
func (r *TransactionRepository) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE transactions SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List returns non-deleted transactions matching the filter, newest first.
func (r *TransactionRepository) List(ctx context.Context, f domain.TransactionFilter) ([]domain.Transaction, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, "$"+strconv.Itoa(len(args))))
	}

	if f.From != nil {
		add("data >= %s", f.From.Time)
	}
	if f.To != nil {
		add("data <= %s", f.To.Time)
	}
	if f.Tipo != nil {
		add("tipo = %s", *f.Tipo)
	}
	if f.ProjectID != nil {
		add("project_id = %s", *f.ProjectID)
	}
	if f.UserID != nil {
		add("user_id = %s", *f.UserID)
	}
	if f.CategoryID != nil {
		add("category_id = %s", *f.CategoryID)
	}
	if f.Origem != nil {
		add("origem = %s", *f.Origem)
	}

	q := `SELECT ` + txColumns + ` FROM transactions WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY data DESC, id DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txs []domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, *t)
	}
	return txs, rows.Err()
}

// mapTxFKError turns a foreign-key violation (bad project/user/category) into a
// validation error the client can act on.
func mapTxFKError(err error) error {
	if isForeignKeyViolation(err) {
		return fmt.Errorf("%w: projeto, colaborador ou categoria inexistente", domain.ErrValidation)
	}
	return err
}
