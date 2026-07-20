package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type AttachmentRepository struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepository(pool *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{pool: pool}
}

// --- comprovantes (attachments) ---

func (r *AttachmentRepository) Create(ctx context.Context, a *domain.Attachment) (*domain.Attachment, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO attachments (owner_type, owner_id, url, nome_arquivo, descricao, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, owner_type, owner_id, url, nome_arquivo, descricao, created_by, created_at`,
		a.OwnerType, a.OwnerID, a.URL, a.NomeArquivo, a.Descricao, a.CreatedBy)
	return scanAttachment(row)
}

func (r *AttachmentRepository) ListByOwner(ctx context.Context, ownerType domain.AttachmentOwner, ownerID int64) ([]domain.Attachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_type, owner_id, url, nome_arquivo, descricao, created_by, created_at
		FROM attachments WHERE owner_type = $1 AND owner_id = $2
		ORDER BY created_at DESC`, ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var out []domain.Attachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// Get returns an attachment by id (used to resolve its storage key on delete).
func (r *AttachmentRepository) Get(ctx context.Context, id int64) (*domain.Attachment, error) {
	return scanAttachment(r.pool.QueryRow(ctx, `
		SELECT id, owner_type, owner_id, url, nome_arquivo, descricao, created_by, created_at
		FROM attachments WHERE id = $1`, id))
}

func (r *AttachmentRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanAttachment(row pgx.Row) (*domain.Attachment, error) {
	var a domain.Attachment
	err := row.Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.URL, &a.NomeArquivo, &a.Descricao, &a.CreatedBy, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// --- propostas (project_proposals) ---

func (r *AttachmentRepository) CreateProposal(ctx context.Context, p *domain.Proposal) (*domain.Proposal, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO project_proposals (project_id, url, arquivo_tipo, nome_arquivo, descricao, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, project_id, url, arquivo_tipo, nome_arquivo, descricao, created_by, created_at`,
		p.ProjectID, p.URL, p.ArquivoTipo, p.NomeArquivo, p.Descricao, p.CreatedBy)
	proposal, err := scanProposal(row)
	if isForeignKeyViolation(err) {
		return nil, domain.ErrNotFound // project doesn't exist
	}
	return proposal, err
}

func (r *AttachmentRepository) ListProposals(ctx context.Context, projectID int64) ([]domain.Proposal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, url, arquivo_tipo, nome_arquivo, descricao, created_by, created_at
		FROM project_proposals WHERE project_id = $1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close()

	var out []domain.Proposal
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *AttachmentRepository) GetProposal(ctx context.Context, id int64) (*domain.Proposal, error) {
	return scanProposal(r.pool.QueryRow(ctx, `
		SELECT id, project_id, url, arquivo_tipo, nome_arquivo, descricao, created_by, created_at
		FROM project_proposals WHERE id = $1`, id))
}

func (r *AttachmentRepository) DeleteProposal(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM project_proposals WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete proposal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanProposal(row pgx.Row) (*domain.Proposal, error) {
	var p domain.Proposal
	err := row.Scan(&p.ID, &p.ProjectID, &p.URL, &p.ArquivoTipo, &p.NomeArquivo, &p.Descricao, &p.CreatedBy, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
