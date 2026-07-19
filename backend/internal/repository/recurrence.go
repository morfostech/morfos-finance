package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type RecurrenceRepository struct {
	pool *pgxpool.Pool
}

func NewRecurrenceRepository(pool *pgxpool.Pool) *RecurrenceRepository {
	return &RecurrenceRepository{pool: pool}
}

// MonthRows returns every project with a monthly fee alongside the recurrence
// income it received in [start, end]. The active-period decision and the
// previsto/pendente math are done in the domain layer (BuildSummary).
func (r *RecurrenceRepository) MonthRows(ctx context.Context, start, end time.Time, projectID *int64) ([]domain.RecurrenceRow, error) {
	q := `
		SELECT p.id, p.nome, p.valor_mensal::text, p.data_inicio, p.data_fim,
		       COALESCE(SUM(t.valor), 0)::text AS recebido
		FROM projects p
		LEFT JOIN transactions t
		  ON t.project_id = p.id
		 AND t.tipo = 'ganho' AND t.origem = 'recorrencia'
		 AND t.deleted_at IS NULL
		 AND t.data >= $1 AND t.data <= $2
		WHERE p.valor_mensal IS NOT NULL`
	args := []any{start, end}
	if projectID != nil {
		q += ` AND p.id = $3`
		args = append(args, *projectID)
	}
	q += ` GROUP BY p.id ORDER BY p.nome`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("recurrence rows: %w", err)
	}
	defer rows.Close()

	var out []domain.RecurrenceRow
	for rows.Next() {
		var (
			row         domain.RecurrenceRow
			valorMensal string
			recebido    string
			inicio      *time.Time
			fim         *time.Time
		)
		if err := rows.Scan(&row.ProjectID, &row.Nome, &valorMensal, &inicio, &fim, &recebido); err != nil {
			return nil, err
		}
		if row.ValorMensal, err = domain.ParseNumeric(valorMensal); err != nil {
			return nil, err
		}
		if row.Recebido, err = domain.ParseNumeric(recebido); err != nil {
			return nil, err
		}
		row.DataInicio = datePtr(inicio)
		row.DataFim = datePtr(fim)
		out = append(out, row)
	}
	return out, rows.Err()
}
