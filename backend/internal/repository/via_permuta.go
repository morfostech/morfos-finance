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

type ViaPermutaRepository struct {
	pool *pgxpool.Pool
}

func NewViaPermutaRepository(pool *pgxpool.Pool) *ViaPermutaRepository {
	return &ViaPermutaRepository{pool: pool}
}

const vpTransactionColumns = `id, tipo, status, valor::text, data, permutante, oferta,
	project_id, voucher_code, observacoes, created_by, created_at, updated_at`

func scanVPTransaction(row pgx.Row) (*domain.VPTransaction, error) {
	var (
		t     domain.VPTransaction
		valor string
		data  time.Time
	)
	if err := row.Scan(&t.ID, &t.Tipo, &t.Status, &valor, &data, &t.Permutante,
		&t.Oferta, &t.ProjectID, &t.VoucherCode, &t.Observacoes, &t.CreatedBy,
		&t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	amount, err := domain.ParseNumeric(valor)
	if err != nil {
		return nil, err
	}
	t.Valor = amount
	t.Data = domain.NewDate(data)
	return &t, nil
}

func (r *ViaPermutaRepository) CreateTransaction(ctx context.Context, t *domain.VPTransaction) (*domain.VPTransaction, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO vp_transactions
			(tipo, status, valor, data, permutante, oferta, project_id, voucher_code, observacoes, created_by)
		VALUES ($1, $2, $3::numeric, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+vpTransactionColumns,
		t.Tipo, t.Status, t.Valor.Numeric(), t.Data.Time, t.Permutante, t.Oferta,
		t.ProjectID, t.VoucherCode, t.Observacoes, t.CreatedBy)
	created, err := scanVPTransaction(row)
	if err != nil {
		return nil, mapVPFKError(err)
	}
	return created, nil
}

func (r *ViaPermutaRepository) GetTransaction(ctx context.Context, id int64) (*domain.VPTransaction, error) {
	return scanVPTransaction(r.pool.QueryRow(ctx,
		`SELECT `+vpTransactionColumns+` FROM vp_transactions WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *ViaPermutaRepository) UpdateTransaction(ctx context.Context, t *domain.VPTransaction) (*domain.VPTransaction, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE vp_transactions SET tipo = $2, status = $3, valor = $4::numeric,
			data = $5, permutante = $6, oferta = $7, project_id = $8,
			voucher_code = $9, observacoes = $10
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+vpTransactionColumns,
		t.ID, t.Tipo, t.Status, t.Valor.Numeric(), t.Data.Time, t.Permutante,
		t.Oferta, t.ProjectID, t.VoucherCode, t.Observacoes)
	updated, err := scanVPTransaction(row)
	if err != nil {
		return nil, mapVPFKError(err)
	}
	return updated, nil
}

func (r *ViaPermutaRepository) DeleteTransaction(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vp_transactions SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete VP transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ViaPermutaRepository) ListTransactions(ctx context.Context, f domain.VPTransactionFilter) ([]domain.VPTransaction, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	add := func(condition string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(condition, "$"+strconv.Itoa(len(args))))
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
	if f.Status != nil {
		add("status = %s", *f.Status)
	}

	rows, err := r.pool.Query(ctx, `SELECT `+vpTransactionColumns+` FROM vp_transactions WHERE `+
		strings.Join(conds, " AND ")+` ORDER BY data DESC, id DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list VP transactions: %w", err)
	}
	defer rows.Close()

	var result []domain.VPTransaction
	for rows.Next() {
		t, err := scanVPTransaction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *t)
	}
	return result, rows.Err()
}

func (r *ViaPermutaRepository) GetSettings(ctx context.Context) (*domain.VPSettings, error) {
	var (
		settings domain.VPSettings
		limit    string
	)
	err := r.pool.QueryRow(ctx,
		`SELECT credit_limit::text, updated_at FROM vp_settings WHERE id = 1`).Scan(&limit, &settings.UpdatedAt)
	if err != nil {
		return nil, err
	}
	settings.CreditLimit, err = domain.ParseNumeric(limit)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *ViaPermutaRepository) UpdateSettings(ctx context.Context, creditLimit domain.Money) (*domain.VPSettings, error) {
	var (
		settings domain.VPSettings
		limit    string
	)
	err := r.pool.QueryRow(ctx, `
		UPDATE vp_settings SET credit_limit = $1::numeric WHERE id = 1
		RETURNING credit_limit::text, updated_at`, creditLimit.Numeric()).Scan(&limit, &settings.UpdatedAt)
	if err != nil {
		return nil, err
	}
	settings.CreditLimit, err = domain.ParseNumeric(limit)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *ViaPermutaRepository) Summary(ctx context.Context, f domain.VPTransactionFilter) (*domain.VPSummary, error) {
	var from, to any
	if f.From != nil {
		from = f.From.Time
	}
	if f.To != nil {
		to = f.To.Time
	}
	row := r.pool.QueryRow(ctx, `
		WITH ledger AS (
			SELECT
				COALESCE(SUM(CASE WHEN tipo = 'venda' AND status = 'concluida' THEN valor ELSE 0 END), 0) AS sales,
				COALESCE(SUM(CASE WHEN tipo = 'compra' AND status = 'concluida' THEN valor ELSE 0 END), 0) AS purchases,
				COUNT(*) FILTER (WHERE status = 'negociando') AS open_deals
			FROM vp_transactions WHERE deleted_at IS NULL
		), period AS (
			SELECT
				COALESCE(SUM(valor) FILTER (WHERE tipo = 'venda' AND status = 'concluida'), 0) AS sales,
				COALESCE(SUM(valor) FILTER (WHERE tipo = 'compra' AND status = 'concluida'), 0) AS purchases,
				COALESCE(ROUND(AVG(valor) FILTER (WHERE tipo = 'venda' AND status = 'concluida'), 2), 0) AS avg_sale,
				COALESCE(ROUND(AVG(valor) FILTER (WHERE tipo = 'compra' AND status = 'concluida'), 2), 0) AS avg_purchase
			FROM vp_transactions
			WHERE deleted_at IS NULL
				AND ($1::date IS NULL OR data >= $1)
				AND ($2::date IS NULL OR data <= $2)
		), offers AS (
			SELECT
				COUNT(*) FILTER (WHERE status = 'aberta') AS open_offers,
				COUNT(*) FILTER (WHERE status = 'liberada') AS released_offers,
				COUNT(*) FILTER (WHERE status IN ('pendente', 'bloqueada')) AS pending_offers
			FROM vp_offers WHERE deleted_at IS NULL
		)
		SELECT (l.sales - l.purchases)::text, s.credit_limit::text,
			(s.credit_limit + l.sales - l.purchases)::text,
			p.sales::text, p.purchases::text, p.avg_sale::text, p.avg_purchase::text,
			l.open_deals, o.open_offers, o.released_offers, o.pending_offers
		FROM ledger l CROSS JOIN period p CROSS JOIN offers o CROSS JOIN vp_settings s
		WHERE s.id = 1`, from, to)

	var (
		summary domain.VPSummary
		values  [7]string
	)
	err := row.Scan(&values[0], &values[1], &values[2], &values[3], &values[4],
		&values[5], &values[6], &summary.NegociacoesAbertas, &summary.OfertasAbertas,
		&summary.OfertasLiberadas, &summary.OfertasComPendencia)
	if err != nil {
		return nil, err
	}
	destinations := []*domain.Money{&summary.Saldo, &summary.LimiteCredito, &summary.Disponivel,
		&summary.VendasPeriodo, &summary.ComprasPeriodo, &summary.TicketMedioVenda, &summary.TicketMedioCompra}
	for i, value := range values {
		amount, err := domain.ParseNumeric(value)
		if err != nil {
			return nil, err
		}
		*destinations[i] = amount
	}
	return &summary, nil
}

const vpOfferColumns = `id, titulo, descricao, valor::text, negociavel, status,
	external_url, created_by, created_at, updated_at`

func scanVPOffer(row pgx.Row) (*domain.VPOffer, error) {
	var (
		o     domain.VPOffer
		valor *string
	)
	if err := row.Scan(&o.ID, &o.Titulo, &o.Descricao, &valor, &o.Negociavel,
		&o.Status, &o.ExternalURL, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if valor != nil {
		amount, err := domain.ParseNumeric(*valor)
		if err != nil {
			return nil, err
		}
		o.Valor = &amount
	}
	return &o, nil
}

func vpAmountValue(value *domain.Money) any {
	if value == nil {
		return nil
	}
	return value.Numeric()
}

func (r *ViaPermutaRepository) CreateOffer(ctx context.Context, o *domain.VPOffer) (*domain.VPOffer, error) {
	return scanVPOffer(r.pool.QueryRow(ctx, `
		INSERT INTO vp_offers (titulo, descricao, valor, negociavel, status, external_url, created_by)
		VALUES ($1, $2, $3::numeric, $4, $5, $6, $7)
		RETURNING `+vpOfferColumns,
		o.Titulo, o.Descricao, vpAmountValue(o.Valor), o.Negociavel, o.Status, o.ExternalURL, o.CreatedBy))
}

func (r *ViaPermutaRepository) GetOffer(ctx context.Context, id int64) (*domain.VPOffer, error) {
	return scanVPOffer(r.pool.QueryRow(ctx,
		`SELECT `+vpOfferColumns+` FROM vp_offers WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *ViaPermutaRepository) UpdateOffer(ctx context.Context, o *domain.VPOffer) (*domain.VPOffer, error) {
	return scanVPOffer(r.pool.QueryRow(ctx, `
		UPDATE vp_offers SET titulo = $2, descricao = $3, valor = $4::numeric,
			negociavel = $5, status = $6, external_url = $7
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+vpOfferColumns,
		o.ID, o.Titulo, o.Descricao, vpAmountValue(o.Valor), o.Negociavel, o.Status, o.ExternalURL))
}

func (r *ViaPermutaRepository) DeleteOffer(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vp_offers SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete VP offer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ViaPermutaRepository) ListOffers(ctx context.Context) ([]domain.VPOffer, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+vpOfferColumns+`
		FROM vp_offers WHERE deleted_at IS NULL ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list VP offers: %w", err)
	}
	defer rows.Close()
	var result []domain.VPOffer
	for rows.Next() {
		o, err := scanVPOffer(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *o)
	}
	return result, rows.Err()
}

func mapVPFKError(err error) error {
	if isForeignKeyViolation(err) {
		return fmt.Errorf("%w: projeto ou usuário inexistente", domain.ErrValidation)
	}
	return err
}
