package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type ChangeRequestRepository struct {
	pool *pgxpool.Pool
}

func NewChangeRequestRepository(pool *pgxpool.Pool) *ChangeRequestRepository {
	return &ChangeRequestRepository{pool: pool}
}

const changeRequestColumns = `
	cr.id, cr.requester_id, requester.nome, cr.action, cr.payload, cr.status,
	cr.reviewer_id, reviewer.nome, cr.review_comment, cr.created_at, cr.reviewed_at`

func scanChangeRequest(row pgx.Row) (*domain.ChangeRequest, error) {
	var cr domain.ChangeRequest
	err := row.Scan(&cr.ID, &cr.RequesterID, &cr.RequesterName, &cr.Action, &cr.Payload,
		&cr.Status, &cr.ReviewerID, &cr.ReviewerName, &cr.ReviewComment, &cr.CreatedAt, &cr.ReviewedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *ChangeRequestRepository) Create(ctx context.Context, cr *domain.ChangeRequest) (*domain.ChangeRequest, error) {
	row := r.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO change_requests (requester_id, action, payload)
			VALUES ($1, $2, $3) RETURNING *
		)
		SELECT `+changeRequestColumns+`
		FROM inserted cr
		JOIN users requester ON requester.id = cr.requester_id
		LEFT JOIN users reviewer ON reviewer.id = cr.reviewer_id`,
		cr.RequesterID, cr.Action, cr.Payload)
	return scanChangeRequest(row)
}

func (r *ChangeRequestRepository) List(ctx context.Context, requesterID *int64) ([]domain.ChangeRequest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+changeRequestColumns+`
		FROM change_requests cr
		JOIN users requester ON requester.id = cr.requester_id
		LEFT JOIN users reviewer ON reviewer.id = cr.reviewer_id
		WHERE ($1::bigint IS NULL OR cr.requester_id = $1)
		  AND cr.status <> 'processing'
		ORDER BY (cr.status = 'pending') DESC, cr.created_at DESC`, requesterID)
	if err != nil {
		return nil, fmt.Errorf("list change requests: %w", err)
	}
	defer rows.Close()

	var out []domain.ChangeRequest
	for rows.Next() {
		cr, err := scanChangeRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *cr)
	}
	return out, rows.Err()
}

// StartReview atomically claims a pending request so two reviewers cannot
// execute the same requested mutation.
func (r *ChangeRequestRepository) StartReview(ctx context.Context, id int64) (*domain.ChangeRequest, error) {
	row := r.pool.QueryRow(ctx, `
		WITH claimed AS (
			UPDATE change_requests SET status = 'processing'
			WHERE id = $1 AND status = 'pending' RETURNING *
		)
		SELECT `+changeRequestColumns+`
		FROM claimed cr
		JOIN users requester ON requester.id = cr.requester_id
		LEFT JOIN users reviewer ON reviewer.id = cr.reviewer_id`, id)
	cr, err := scanChangeRequest(row)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("%w: solicitação já foi revisada", domain.ErrConflict)
	}
	return cr, err
}

func (r *ChangeRequestRepository) FinishReview(ctx context.Context, id, reviewerID int64, status domain.ChangeRequestStatus, comment *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE change_requests
		SET status = $3, reviewer_id = $2, review_comment = $4, reviewed_at = now()
		WHERE id = $1 AND status = 'processing'`, id, reviewerID, status, comment)
	if err != nil {
		return fmt.Errorf("finish change request review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (r *ChangeRequestRepository) RestorePending(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE change_requests SET status = 'pending' WHERE id = $1 AND status = 'processing'`, id)
	return err
}
