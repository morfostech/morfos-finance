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

type PlanningRepository struct{ pool *pgxpool.Pool }

func NewPlanningRepository(pool *pgxpool.Pool) *PlanningRepository {
	return &PlanningRepository{pool: pool}
}

const plannedColumns = `id, tipo, valor::text, due_date, project_id, user_id, origem,
	category_id, descricao, actual_transaction_id, created_by, created_at, updated_at`

func scanPlanned(row pgx.Row) (*domain.PlannedEntry, error) {
	var p domain.PlannedEntry
	var valor string
	var due time.Time
	if err := row.Scan(&p.ID, &p.Tipo, &valor, &due, &p.ProjectID, &p.UserID,
		&p.Origem, &p.CategoryID, &p.Descricao, &p.ActualTransactionID,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	money, err := domain.ParseNumeric(valor)
	if err != nil {
		return nil, err
	}
	p.Valor = money
	p.DueDate = domain.NewDate(due)
	return &p, nil
}

func (r *PlanningRepository) CreateMany(ctx context.Context, entries []domain.PlannedEntry) ([]domain.PlannedEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]domain.PlannedEntry, 0, len(entries))
	for _, p := range entries {
		created, err := scanPlanned(tx.QueryRow(ctx, `
			INSERT INTO planned_entries
				(tipo, valor, due_date, project_id, user_id, origem, category_id, descricao, created_by)
			VALUES ($1, $2::numeric, $3, $4, $5, $6, $7, $8, $9)
			RETURNING `+plannedColumns,
			p.Tipo, p.Valor.Numeric(), p.DueDate.Time, p.ProjectID, p.UserID,
			p.Origem, p.CategoryID, p.Descricao, p.CreatedBy))
		if err != nil {
			return nil, mapTxFKError(err)
		}
		out = append(out, *created)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PlanningRepository) GetByID(ctx context.Context, id int64) (*domain.PlannedEntry, error) {
	return scanPlanned(r.pool.QueryRow(ctx, `SELECT `+plannedColumns+`
		FROM planned_entries WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *PlanningRepository) Update(ctx context.Context, p *domain.PlannedEntry) (*domain.PlannedEntry, error) {
	updated, err := scanPlanned(r.pool.QueryRow(ctx, `
		UPDATE planned_entries SET tipo=$2, valor=$3::numeric, due_date=$4, project_id=$5,
			user_id=$6, origem=$7, category_id=$8, descricao=$9
		WHERE id=$1 AND deleted_at IS NULL AND actual_transaction_id IS NULL
		RETURNING `+plannedColumns,
		p.ID, p.Tipo, p.Valor.Numeric(), p.DueDate.Time, p.ProjectID, p.UserID,
		p.Origem, p.CategoryID, p.Descricao))
	if err != nil {
		return nil, mapTxFKError(err)
	}
	return updated, nil
}

func (r *PlanningRepository) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE planned_entries SET deleted_at=now()
		WHERE id=$1 AND deleted_at IS NULL AND actual_transaction_id IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PlanningRepository) List(ctx context.Context, f domain.PlanningFilter) ([]domain.PlannedEntry, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	add := func(template string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(template, "$"+strconv.Itoa(len(args))))
	}
	if f.From != nil {
		add("due_date >= %s", f.From.Time)
	}
	if f.To != nil {
		add("due_date <= %s", f.To.Time)
	}
	if f.Status != nil {
		if *f.Status == domain.PlannedOpen {
			conds = append(conds, "actual_transaction_id IS NULL")
		} else {
			conds = append(conds, "actual_transaction_id IS NOT NULL")
		}
	}
	rows, err := r.pool.Query(ctx, `SELECT `+plannedColumns+` FROM planned_entries WHERE `+
		strings.Join(conds, " AND ")+` ORDER BY due_date, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PlannedEntry
	for rows.Next() {
		p, err := scanPlanned(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PlanningRepository) Complete(ctx context.Context, id, completedBy int64, paidOn domain.Date) (*domain.PlannedEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	p, err := scanPlanned(tx.QueryRow(ctx, `SELECT `+plannedColumns+`
		FROM planned_entries WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id))
	if err != nil {
		return nil, err
	}
	if p.ActualTransactionID != nil {
		return nil, fmt.Errorf("%w: lançamento já realizado", domain.ErrConflict)
	}
	var txID int64
	err = tx.QueryRow(ctx, `INSERT INTO transactions
		(tipo, valor, data, project_id, user_id, origem, category_id, descricao, created_by)
		VALUES ($1,$2::numeric,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		p.Tipo, p.Valor.Numeric(), paidOn.Time, p.ProjectID, p.UserID, p.Origem,
		p.CategoryID, p.Descricao, completedBy).Scan(&txID)
	if err != nil {
		return nil, mapTxFKError(err)
	}
	p, err = scanPlanned(tx.QueryRow(ctx, `UPDATE planned_entries SET actual_transaction_id=$2
		WHERE id=$1 RETURNING `+plannedColumns, id, txID))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PlanningRepository) OpeningBalance(ctx context.Context, before time.Time) (domain.Money, error) {
	var value string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN tipo='ganho' THEN valor ELSE -valor END),0)::text
		FROM transactions WHERE deleted_at IS NULL AND data < $1`, before).Scan(&value)
	if err != nil {
		return 0, err
	}
	return domain.ParseNumeric(value)
}

func (r *PlanningRepository) CountOverdue(ctx context.Context, today time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM planned_entries
		WHERE deleted_at IS NULL AND actual_transaction_id IS NULL AND due_date < $1`, today).Scan(&count)
	return count, err
}

func (r *PlanningRepository) UpsertBudget(ctx context.Context, categoryID int64, year, month int, value domain.Money, createdBy int64) (*domain.ExpenseBudget, error) {
	var b domain.ExpenseBudget
	var budget, actual string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO expense_budgets (category_id, ano, mes, valor, created_by)
		VALUES ($1,$2,$3,$4::numeric,$5)
		ON CONFLICT (category_id,ano,mes) DO UPDATE SET valor=EXCLUDED.valor
		RETURNING id, category_id, (SELECT nome FROM expense_categories WHERE id=$1), ano, mes,
			valor::text, COALESCE((SELECT SUM(t.valor) FROM transactions t
			WHERE t.deleted_at IS NULL AND t.tipo='despesa' AND t.category_id=$1
			AND EXTRACT(YEAR FROM t.data)=$2 AND EXTRACT(MONTH FROM t.data)=$3),0)::text`,
		categoryID, year, month, value.Numeric(), createdBy).
		Scan(&b.ID, &b.CategoryID, &b.Category, &b.Ano, &b.Mes, &budget, &actual)
	if err != nil {
		return nil, mapTxFKError(err)
	}
	return finishBudget(&b, budget, actual)
}

func (r *PlanningRepository) ListBudgets(ctx context.Context, year, month int) ([]domain.ExpenseBudget, error) {
	rows, err := r.pool.Query(ctx, `SELECT b.id,b.category_id,c.nome,b.ano,b.mes,b.valor::text,
		COALESCE(SUM(t.valor),0)::text
		FROM expense_budgets b JOIN expense_categories c ON c.id=b.category_id
		LEFT JOIN transactions t ON t.category_id=b.category_id AND t.deleted_at IS NULL
			AND t.tipo='despesa' AND EXTRACT(YEAR FROM t.data)=b.ano AND EXTRACT(MONTH FROM t.data)=b.mes
		WHERE b.ano=$1 AND b.mes=$2 GROUP BY b.id,c.nome ORDER BY c.nome`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ExpenseBudget
	for rows.Next() {
		var b domain.ExpenseBudget
		var budget, actual string
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Category, &b.Ano, &b.Mes, &budget, &actual); err != nil {
			return nil, err
		}
		finished, err := finishBudget(&b, budget, actual)
		if err != nil {
			return nil, err
		}
		out = append(out, *finished)
	}
	return out, rows.Err()
}

func finishBudget(b *domain.ExpenseBudget, budget, actual string) (*domain.ExpenseBudget, error) {
	var err error
	if b.Valor, err = domain.ParseNumeric(budget); err != nil {
		return nil, err
	}
	if b.Realizado, err = domain.ParseNumeric(actual); err != nil {
		return nil, err
	}
	b.Restante = b.Valor - b.Realizado
	if b.Valor > 0 {
		b.Percentual = int((b.Realizado * 100) / b.Valor)
	}
	return b, nil
}

func (r *PlanningRepository) DeleteBudget(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM expense_budgets WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
