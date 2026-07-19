package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

// SaldoEmCaixa is the all-time accumulated balance: sum(ganhos) - sum(despesas)
// over non-deleted transactions.
func (r *DashboardRepository) SaldoEmCaixa(ctx context.Context) (domain.Money, error) {
	var s string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN tipo = 'ganho' THEN valor ELSE -valor END), 0)::text
		FROM transactions WHERE deleted_at IS NULL`).Scan(&s)
	if err != nil {
		return 0, fmt.Errorf("saldo em caixa: %w", err)
	}
	return domain.ParseNumeric(s)
}

// PeriodTotals returns income and expense sums within [from, to].
func (r *DashboardRepository) PeriodTotals(ctx context.Context, from, to time.Time) (ganhos, despesas domain.Money, err error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tipo, COALESCE(SUM(valor), 0)::text
		FROM transactions
		WHERE deleted_at IS NULL AND data >= $1 AND data <= $2
		GROUP BY tipo`, from, to)
	if err != nil {
		return 0, 0, fmt.Errorf("period totals: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tipo domain.TxType
		var val string
		if err := rows.Scan(&tipo, &val); err != nil {
			return 0, 0, err
		}
		m, err := domain.ParseNumeric(val)
		if err != nil {
			return 0, 0, err
		}
		switch tipo {
		case domain.TxGanho:
			ganhos = m
		case domain.TxDespesa:
			despesas = m
		}
	}
	return ganhos, despesas, rows.Err()
}

// GanhosPorOrigem splits income by origem within [from, to].
func (r *DashboardRepository) GanhosPorOrigem(ctx context.Context, from, to time.Time) (domain.OrigemTotals, error) {
	var out domain.OrigemTotals
	rows, err := r.pool.Query(ctx, `
		SELECT origem, COALESCE(SUM(valor), 0)::text
		FROM transactions
		WHERE deleted_at IS NULL AND tipo = 'ganho' AND data >= $1 AND data <= $2
		GROUP BY origem`, from, to)
	if err != nil {
		return out, fmt.Errorf("ganhos por origem: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var origem *domain.TxOrigem
		var val string
		if err := rows.Scan(&origem, &val); err != nil {
			return out, err
		}
		m, err := domain.ParseNumeric(val)
		if err != nil {
			return out, err
		}
		switch {
		case origem == nil:
			out.SemOrigem = m
		case *origem == domain.OrigemImplementacao:
			out.Implementacao = m
		case *origem == domain.OrigemRecorrencia:
			out.Recorrencia = m
		case *origem == domain.OrigemAvulso:
			out.Avulso = m
		}
	}
	return out, rows.Err()
}

// DespesasPorCategoria groups expenses by category (nil -> "Sem categoria").
func (r *DashboardRepository) DespesasPorCategoria(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, COALESCE(c.nome, 'Sem categoria'), COALESCE(SUM(t.valor), 0)::text
		FROM transactions t
		LEFT JOIN expense_categories c ON c.id = t.category_id
		WHERE t.deleted_at IS NULL AND t.tipo = 'despesa' AND t.data >= $1 AND t.data <= $2
		GROUP BY c.id, c.nome
		ORDER BY SUM(t.valor) DESC`, from, to)
	if err != nil {
		return nil, fmt.Errorf("despesas por categoria: %w", err)
	}
	defer rows.Close()

	var out []domain.CategoryTotal
	for rows.Next() {
		var ct domain.CategoryTotal
		var val string
		if err := rows.Scan(&ct.CategoryID, &ct.Nome, &val); err != nil {
			return nil, err
		}
		if ct.Total, err = domain.ParseNumeric(val); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

// ImplementacaoTotals returns the all-time installment totals.
func (r *DashboardRepository) ImplementacaoTotals(ctx context.Context) (domain.ImplementacaoTotals, error) {
	var out domain.ImplementacaoTotals
	var total, recebido, aReceber string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(valor), 0)::text,
		       COALESCE(SUM(valor) FILTER (WHERE pago_em IS NOT NULL), 0)::text,
		       COALESCE(SUM(valor) FILTER (WHERE pago_em IS NULL), 0)::text
		FROM project_installments`).Scan(&total, &recebido, &aReceber)
	if err != nil {
		return out, fmt.Errorf("implementacao totals: %w", err)
	}
	if out.Total, err = domain.ParseNumeric(total); err != nil {
		return out, err
	}
	if out.Recebido, err = domain.ParseNumeric(recebido); err != nil {
		return out, err
	}
	out.AReceber, err = domain.ParseNumeric(aReceber)
	return out, err
}

// ParcelasPendentes counts and sums unpaid installments.
func (r *DashboardRepository) ParcelasPendentes(ctx context.Context) (domain.PendingInstallments, error) {
	var out domain.PendingInstallments
	var total string
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(valor), 0)::text
		FROM project_installments WHERE pago_em IS NULL`).Scan(&out.Quantidade, &total)
	if err != nil {
		return out, fmt.Errorf("parcelas pendentes: %w", err)
	}
	out.Total, err = domain.ParseNumeric(total)
	return out, err
}

// PorProjeto returns income/expense per project within [from, to].
func (r *DashboardRepository) PorProjeto(ctx context.Context, from, to time.Time) ([]domain.ProjectTotals, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.nome,
		       COALESCE(SUM(t.valor) FILTER (WHERE t.tipo = 'ganho'), 0)::text,
		       COALESCE(SUM(t.valor) FILTER (WHERE t.tipo = 'despesa'), 0)::text
		FROM projects p
		JOIN transactions t ON t.project_id = p.id
		WHERE t.deleted_at IS NULL AND t.data >= $1 AND t.data <= $2
		GROUP BY p.id, p.nome
		ORDER BY p.nome`, from, to)
	if err != nil {
		return nil, fmt.Errorf("por projeto: %w", err)
	}
	defer rows.Close()

	var out []domain.ProjectTotals
	for rows.Next() {
		var pt domain.ProjectTotals
		var g, d string
		if err := rows.Scan(&pt.ProjectID, &pt.Nome, &g, &d); err != nil {
			return nil, err
		}
		if pt.Ganhos, err = domain.ParseNumeric(g); err != nil {
			return nil, err
		}
		if pt.Despesas, err = domain.ParseNumeric(d); err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, rows.Err()
}

// PorColaborador returns income/expense per collaborator within [from, to].
func (r *DashboardRepository) PorColaborador(ctx context.Context, from, to time.Time) ([]domain.UserTotals, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.nome,
		       COALESCE(SUM(t.valor) FILTER (WHERE t.tipo = 'ganho'), 0)::text,
		       COALESCE(SUM(t.valor) FILTER (WHERE t.tipo = 'despesa'), 0)::text
		FROM users u
		JOIN transactions t ON t.user_id = u.id
		WHERE t.deleted_at IS NULL AND t.data >= $1 AND t.data <= $2
		GROUP BY u.id, u.nome
		ORDER BY u.nome`, from, to)
	if err != nil {
		return nil, fmt.Errorf("por colaborador: %w", err)
	}
	defer rows.Close()

	var out []domain.UserTotals
	for rows.Next() {
		var ut domain.UserTotals
		var g, d string
		if err := rows.Scan(&ut.UserID, &ut.Nome, &g, &d); err != nil {
			return nil, err
		}
		if ut.Ganhos, err = domain.ParseNumeric(g); err != nil {
			return nil, err
		}
		if ut.Despesas, err = domain.ParseNumeric(d); err != nil {
			return nil, err
		}
		out = append(out, ut)
	}
	return out, rows.Err()
}

// UserTotalsFor returns income/expense for a single collaborator within [from, to].
func (r *DashboardRepository) UserTotalsFor(ctx context.Context, userID int64, from, to time.Time) (ganhos, despesas domain.Money, err error) {
	var g, d string
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(valor) FILTER (WHERE tipo = 'ganho'), 0)::text,
		       COALESCE(SUM(valor) FILTER (WHERE tipo = 'despesa'), 0)::text
		FROM transactions
		WHERE deleted_at IS NULL AND user_id = $1 AND data >= $2 AND data <= $3`,
		userID, from, to).Scan(&g, &d)
	if err != nil {
		return 0, 0, fmt.Errorf("user totals: %w", err)
	}
	if ganhos, err = domain.ParseNumeric(g); err != nil {
		return 0, 0, err
	}
	despesas, err = domain.ParseNumeric(d)
	return ganhos, despesas, err
}
