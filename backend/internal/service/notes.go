package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// NoteRepo is the persistence contract for private notes. All operations are
// scoped to a userID at the repository level so a user can never read, edit,
// or delete another user's notes — not even admin/sócio.
type NoteRepo interface {
	Create(ctx context.Context, n *domain.Note) (*domain.Note, error)
	List(ctx context.Context, userID int64, ownerType domain.NoteOwner, ownerID *int64) ([]domain.Note, error)
	GetOwned(ctx context.Context, id, userID int64) (*domain.Note, error)
	Update(ctx context.Context, id, userID int64, texto string) (*domain.Note, error)
	Delete(ctx context.Context, id, userID int64) error
}

type NoteService struct {
	notes NoteRepo
}

func NewNoteService(notes NoteRepo) *NoteService {
	return &NoteService{notes: notes}
}

func (s *NoteService) ValidateCreate(ownerType domain.NoteOwner, ownerID *int64, texto string) error {
	if !ownerType.Valid() {
		return fmt.Errorf("%w: tipo de anotação inválido", domain.ErrValidation)
	}
	if ownerType != domain.NoteOwnerGeral && ownerID == nil {
		return fmt.Errorf("%w: informe o item ao qual a anotação se refere", domain.ErrValidation)
	}
	return s.ValidateText(texto)
}

func (s *NoteService) ValidateText(texto string) error {
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("%w: anotação vazia", domain.ErrValidation)
	}
	return nil
}

func (s *NoteService) GetOwned(ctx context.Context, id, userID int64) (*domain.Note, error) {
	return s.notes.GetOwned(ctx, id, userID)
}

func (s *NoteService) Create(ctx context.Context, userID int64, ownerType domain.NoteOwner, ownerID *int64, texto string) (*domain.Note, error) {
	if err := s.ValidateCreate(ownerType, ownerID, texto); err != nil {
		return nil, err
	}
	texto = strings.TrimSpace(texto)
	return s.notes.Create(ctx, &domain.Note{
		UserID:    userID,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Texto:     texto,
	})
}

func (s *NoteService) List(ctx context.Context, userID int64, ownerType domain.NoteOwner, ownerID *int64) ([]domain.Note, error) {
	if !ownerType.Valid() {
		return nil, fmt.Errorf("%w: tipo de anotação inválido", domain.ErrValidation)
	}
	return s.notes.List(ctx, userID, ownerType, ownerID)
}

func (s *NoteService) Update(ctx context.Context, id, userID int64, texto string) (*domain.Note, error) {
	if err := s.ValidateText(texto); err != nil {
		return nil, err
	}
	texto = strings.TrimSpace(texto)
	if _, err := s.notes.GetOwned(ctx, id, userID); err != nil {
		return nil, err
	}
	return s.notes.Update(ctx, id, userID, texto)
}

func (s *NoteService) Delete(ctx context.Context, id, userID int64) error {
	return s.notes.Delete(ctx, id, userID)
}
