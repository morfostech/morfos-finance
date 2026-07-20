package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// InstallmentPlan is the outcome of reconciling implementation installments on
// update: whether to wipe existing rows and which new ones to insert.
type InstallmentPlan struct {
	DeleteAll bool
	Create    []domain.Installment
}

const projectColumns = `id, nome, cliente, valor_implementacao::text, valor_mensal::text,
	dia_vencimento, data_inicio, data_fim, status, created_at, updated_at`

// projectRow holds the raw scan targets before conversion to domain types.
type projectRow struct {
	id         int64
	nome       string
	cliente    *string
	valorImpl  *string
	valorMens  *string
	diaVenc    *int32
	dataInicio *time.Time
	dataFim    *time.Time
	status     domain.ProjectStatus
	createdAt  time.Time
	updatedAt  time.Time
}

func scanProject(row pgx.Row) (*domain.Project, error) {
	var r projectRow
	err := row.Scan(&r.id, &r.nome, &r.cliente, &r.valorImpl, &r.valorMens,
		&r.diaVenc, &r.dataInicio, &r.dataFim, &r.status, &r.createdAt, &r.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain()
}

func (r projectRow) toDomain() (*domain.Project, error) {
	p := &domain.Project{
		ID:        r.id,
		Nome:      r.nome,
		Cliente:   r.cliente,
		Status:    r.status,
		CreatedAt: r.createdAt,
		UpdatedAt: r.updatedAt,
	}
	var err error
	if p.ValorImplementacao, err = moneyPtr(r.valorImpl); err != nil {
		return nil, err
	}
	if p.ValorMensal, err = moneyPtr(r.valorMens); err != nil {
		return nil, err
	}
	if r.diaVenc != nil {
		d := int(*r.diaVenc)
		p.DiaVencimento = &d
	}
	p.DataInicio = datePtr(r.dataInicio)
	p.DataFim = datePtr(r.dataFim)
	return p, nil
}

// Create inserts the project, its installments and member links atomically.
func (r *ProjectRepository) Create(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	var created *domain.Project
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO projects
				(nome, cliente, valor_implementacao, valor_mensal, dia_vencimento, data_inicio, data_fim, status)
			VALUES ($1, $2, $3::numeric, $4::numeric, $5, $6, $7, $8)
			RETURNING `+projectColumns,
			p.Nome, p.Cliente, numericArg(p.ValorImplementacao), numericArg(p.ValorMensal),
			p.DiaVencimento, dateArg(p.DataInicio), dateArg(p.DataFim), p.Status)
		var err error
		created, err = scanProject(row)
		if err != nil {
			return err
		}
		if err := insertInstallments(ctx, tx, created.ID, p.Installments); err != nil {
			return err
		}
		return replaceMembers(ctx, tx, created.ID, p.MemberIDs)
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, created.ID)
}

// Update writes scalar fields and applies the installment reconciliation plan
// atomically. Members are managed separately via ReplaceMembers.
func (r *ProjectRepository) Update(ctx context.Context, p *domain.Project, plan InstallmentPlan) (*domain.Project, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE projects SET
				nome = $2, cliente = $3, valor_implementacao = $4::numeric, valor_mensal = $5::numeric,
				dia_vencimento = $6, data_inicio = $7, data_fim = $8, status = $9
			WHERE id = $1`,
			p.ID, p.Nome, p.Cliente, numericArg(p.ValorImplementacao), numericArg(p.ValorMensal),
			p.DiaVencimento, dateArg(p.DataInicio), dateArg(p.DataFim), p.Status)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		if plan.DeleteAll {
			if _, err := tx.Exec(ctx, `DELETE FROM project_installments WHERE project_id = $1`, p.ID); err != nil {
				return err
			}
		}
		return insertInstallments(ctx, tx, p.ID, plan.Create)
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, p.ID)
}

// GetByID returns the project hydrated with installments and member IDs.
func (r *ProjectRepository) GetByID(ctx context.Context, id int64) (*domain.Project, error) {
	p, err := scanProject(r.pool.QueryRow(ctx, `SELECT `+projectColumns+` FROM projects WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}
	if p.Installments, err = r.installments(ctx, id); err != nil {
		return nil, err
	}
	if p.MemberIDs, err = r.memberIDs(ctx, id); err != nil {
		return nil, err
	}
	return p, nil
}

// List returns projects ordered by newest. When memberID is non-nil, only
// projects that user is allocated to are returned.
func (r *ProjectRepository) List(ctx context.Context, memberID *int64) ([]domain.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects`
	var args []any
	if memberID != nil {
		q += ` WHERE id IN (SELECT project_id FROM project_members WHERE user_id = $1)`
		args = append(args, *memberID)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

// ReplaceMembers sets the exact member list for a project.
func (r *ProjectRepository) ReplaceMembers(ctx context.Context, projectID int64, userIDs []int64) error {
	if err := r.ensureExists(ctx, projectID); err != nil {
		return err
	}
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		return replaceMembers(ctx, tx, projectID, userIDs)
	})
	if isForeignKeyViolation(err) {
		return fmt.Errorf("%w: colaborador inexistente", domain.ErrValidation)
	}
	return err
}

// IsMember reports whether a user is allocated to a project.
func (r *ProjectRepository) IsMember(ctx context.Context, projectID, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`,
		projectID, userID).Scan(&exists)
	return exists, err
}

// SetInstallment writes the full desired state of one installment: its value
// and its paid date (nil pagoEm = pending).
func (r *ProjectRepository) SetInstallment(ctx context.Context, projectID, installmentID int64, valor domain.Money, pagoEm *domain.Date, actorID int64) (*domain.Installment, error) {
	var inst *domain.Installment
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE project_installments
			SET valor = $3::numeric, pago_em = $4
			WHERE id = $1 AND project_id = $2
			RETURNING id, project_id, tipo, valor::text, pago_em, created_at, updated_at`,
			installmentID, projectID, valor.Numeric(), dateArg(pagoEm))
		var err error
		inst, err = scanInstallment(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}

		if pagoEm == nil {
			_, err = tx.Exec(ctx, `
				UPDATE transactions
				SET deleted_at = now()
				WHERE installment_id = $1 AND deleted_at IS NULL`, installmentID)
			return err
		}

		description := "Parcela de implementação (entrada)"
		if inst.Tipo == domain.InstallmentFinalizacao {
			description = "Parcela de implementação (finalização)"
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO transactions (
				tipo, valor, data, project_id, origem, descricao, created_by, installment_id
			)
			VALUES ('ganho', $1::numeric, $2, $3, 'implementacao', $4, $5, $6)
			ON CONFLICT (installment_id) WHERE installment_id IS NOT NULL
			DO UPDATE SET
				tipo = 'ganho', valor = EXCLUDED.valor, data = EXCLUDED.data,
				project_id = EXCLUDED.project_id, user_id = NULL,
				origem = 'implementacao', category_id = NULL,
				descricao = EXCLUDED.descricao, deleted_at = NULL`,
			valor.Numeric(), pagoEm.Time, projectID, description, actorID, installmentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetInstallment returns a single installment scoped to its project.
func (r *ProjectRepository) GetInstallment(ctx context.Context, projectID, installmentID int64) (*domain.Installment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, tipo, valor::text, pago_em, created_at, updated_at
		FROM project_installments WHERE id = $1 AND project_id = $2`,
		installmentID, projectID)
	inst, err := scanInstallment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return inst, err
}

// --- internal helpers ---

func (r *ProjectRepository) installments(ctx context.Context, projectID int64) ([]domain.Installment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, tipo, valor::text, pago_em, created_at, updated_at
		FROM project_installments WHERE project_id = $1
		ORDER BY tipo`, projectID)
	if err != nil {
		return nil, fmt.Errorf("load installments: %w", err)
	}
	defer rows.Close()

	var out []domain.Installment
	for rows.Next() {
		inst, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inst)
	}
	return out, rows.Err()
}

func (r *ProjectRepository) memberIDs(ctx context.Context, projectID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM project_members WHERE project_id = $1 ORDER BY user_id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("load members: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ProjectRepository) ensureExists(ctx context.Context, id int64) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}
	return nil
}

func scanInstallment(row pgx.Row) (*domain.Installment, error) {
	var (
		inst   domain.Installment
		valor  string
		pagoEm *time.Time
	)
	if err := row.Scan(&inst.ID, &inst.ProjectID, &inst.Tipo, &valor, &pagoEm,
		&inst.CreatedAt, &inst.UpdatedAt); err != nil {
		return nil, err
	}
	m, err := domain.ParseNumeric(valor)
	if err != nil {
		return nil, err
	}
	inst.Valor = m
	inst.PagoEm = datePtr(pagoEm)
	inst.Pago = pagoEm != nil
	return &inst, nil
}

func insertInstallments(ctx context.Context, tx pgx.Tx, projectID int64, installments []domain.Installment) error {
	for _, inst := range installments {
		if _, err := tx.Exec(ctx, `
			INSERT INTO project_installments (project_id, tipo, valor, pago_em)
			VALUES ($1, $2, $3::numeric, $4)`,
			projectID, inst.Tipo, inst.Valor.Numeric(), dateArg(inst.PagoEm)); err != nil {
			return fmt.Errorf("insert installment: %w", err)
		}
	}
	return nil
}

func replaceMembers(ctx context.Context, tx pgx.Tx, projectID int64, userIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1`, projectID); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO project_members (project_id, user_id) VALUES ($1, $2)`,
			projectID, uid); err != nil {
			return err
		}
	}
	return nil
}

// moneyPtr converts an optional NUMERIC text into an optional Money.
func moneyPtr(s *string) (*domain.Money, error) {
	if s == nil {
		return nil, nil
	}
	m, err := domain.ParseNumeric(*s)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// datePtr wraps an optional time.Time as an optional domain.Date.
func datePtr(t *time.Time) *domain.Date {
	if t == nil {
		return nil
	}
	d := domain.NewDate(*t)
	return &d
}

// numericArg turns an optional Money into a SQL arg (nil -> NULL).
func numericArg(m *domain.Money) any {
	if m == nil {
		return nil
	}
	return m.Numeric()
}

// dateArg turns an optional Date into a SQL arg (nil -> NULL).
func dateArg(d *domain.Date) any {
	if d == nil {
		return nil
	}
	return d.Time
}
