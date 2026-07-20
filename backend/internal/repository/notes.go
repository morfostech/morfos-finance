package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type NoteRepository struct {
	pool *pgxpool.Pool
}

func NewNoteRepository(pool *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{pool: pool}
}

const noteColumns = `id, user_id, owner_type, owner_id, texto, created_at, updated_at`

func scanNote(row pgx.Row) (*domain.Note, error) {
	var n domain.Note
	err := row.Scan(&n.ID, &n.UserID, &n.OwnerType, &n.OwnerID, &n.Texto, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepository) Create(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO notes (user_id, owner_type, owner_id, texto)
		VALUES ($1, $2, $3, $4)
		RETURNING `+noteColumns,
		n.UserID, n.OwnerType, n.OwnerID, n.Texto)
	return scanNote(row)
}

// List returns a user's own notes for the given owner. When ownerID is nil,
// matches "geral" notes (no specific record).
func (r *NoteRepository) List(ctx context.Context, userID int64, ownerType domain.NoteOwner, ownerID *int64) ([]domain.Note, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+noteColumns+`
		FROM notes
		WHERE user_id = $1 AND owner_type = $2 AND owner_id IS NOT DISTINCT FROM $3
		ORDER BY created_at DESC`,
		userID, ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var out []domain.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

// GetOwned returns a note only if it belongs to userID; otherwise ErrNotFound
// (a foreign note is indistinguishable from a missing one).
func (r *NoteRepository) GetOwned(ctx context.Context, id, userID int64) (*domain.Note, error) {
	return scanNote(r.pool.QueryRow(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE id = $1 AND user_id = $2`, id, userID))
}

func (r *NoteRepository) Update(ctx context.Context, id, userID int64, texto string) (*domain.Note, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE notes SET texto = $3
		WHERE id = $1 AND user_id = $2
		RETURNING `+noteColumns,
		id, userID, texto)
	return scanNote(row)
}

func (r *NoteRepository) Delete(ctx context.Context, id, userID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
