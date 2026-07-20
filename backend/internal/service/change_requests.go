package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type ChangeRequestRepo interface {
	Create(context.Context, *domain.ChangeRequest) (*domain.ChangeRequest, error)
	List(context.Context, *int64) ([]domain.ChangeRequest, error)
	StartReview(context.Context, int64) (*domain.ChangeRequest, error)
	FinishReview(context.Context, int64, int64, domain.ChangeRequestStatus, *string) error
	RestorePending(context.Context, int64) error
}

type NoteChangePayload struct {
	NoteID    *int64           `json:"note_id,omitempty"`
	OwnerType domain.NoteOwner `json:"owner_type,omitempty"`
	OwnerID   *int64           `json:"owner_id,omitempty"`
	Texto     string           `json:"texto,omitempty"`
}

type ChangeRequestService struct {
	requests ChangeRequestRepo
	notes    *NoteService
}

func NewChangeRequestService(requests ChangeRequestRepo, notes *NoteService) *ChangeRequestService {
	return &ChangeRequestService{requests: requests, notes: notes}
}

func (s *ChangeRequestService) Create(ctx context.Context, requester Viewer, action domain.ChangeRequestAction, payload NoteChangePayload) (*domain.ChangeRequest, error) {
	if requester.Role != domain.RoleColaborador {
		return nil, fmt.Errorf("%w: admin e sócio podem alterar diretamente", domain.ErrValidation)
	}
	if !action.Valid() {
		return nil, fmt.Errorf("%w: tipo de solicitação inválido", domain.ErrValidation)
	}
	if err := s.validateNoteChange(ctx, requester.UserID, action, payload); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return s.requests.Create(ctx, &domain.ChangeRequest{
		RequesterID: requester.UserID,
		Action:      action,
		Payload:     raw,
		Status:      domain.ChangePending,
	})
}

func (s *ChangeRequestService) List(ctx context.Context, viewer Viewer) ([]domain.ChangeRequest, error) {
	var requesterID *int64
	if viewer.Role == domain.RoleColaborador {
		requesterID = &viewer.UserID
	}
	return s.requests.List(ctx, requesterID)
}

func (s *ChangeRequestService) Approve(ctx context.Context, id int64, reviewer Viewer, comment string) error {
	if reviewer.Role != domain.RoleAdmin && reviewer.Role != domain.RoleSocio {
		return domain.ErrForbidden
	}
	cr, err := s.requests.StartReview(ctx, id)
	if err != nil {
		return err
	}
	if err := s.apply(ctx, cr); err != nil {
		_ = s.requests.RestorePending(ctx, id)
		return err
	}
	return s.requests.FinishReview(ctx, id, reviewer.UserID, domain.ChangeApproved, trimPtr(&comment))
}

func (s *ChangeRequestService) Reject(ctx context.Context, id int64, reviewer Viewer, comment string) error {
	if reviewer.Role != domain.RoleAdmin && reviewer.Role != domain.RoleSocio {
		return domain.ErrForbidden
	}
	if strings.TrimSpace(comment) == "" {
		return fmt.Errorf("%w: informe o motivo da rejeição", domain.ErrValidation)
	}
	if _, err := s.requests.StartReview(ctx, id); err != nil {
		return err
	}
	return s.requests.FinishReview(ctx, id, reviewer.UserID, domain.ChangeRejected, trimPtr(&comment))
}

func (s *ChangeRequestService) validateNoteChange(ctx context.Context, userID int64, action domain.ChangeRequestAction, p NoteChangePayload) error {
	switch action {
	case domain.ChangeNoteCreate:
		return s.notes.ValidateCreate(p.OwnerType, p.OwnerID, p.Texto)
	case domain.ChangeNoteUpdate:
		if p.NoteID == nil {
			return fmt.Errorf("%w: nota é obrigatória", domain.ErrValidation)
		}
		if err := s.notes.ValidateText(p.Texto); err != nil {
			return err
		}
		_, err := s.notes.GetOwned(ctx, *p.NoteID, userID)
		return err
	case domain.ChangeNoteDelete:
		if p.NoteID == nil {
			return fmt.Errorf("%w: nota é obrigatória", domain.ErrValidation)
		}
		_, err := s.notes.GetOwned(ctx, *p.NoteID, userID)
		return err
	default:
		return fmt.Errorf("%w: tipo de solicitação inválido", domain.ErrValidation)
	}
}

func (s *ChangeRequestService) apply(ctx context.Context, cr *domain.ChangeRequest) error {
	var p NoteChangePayload
	if err := json.Unmarshal(cr.Payload, &p); err != nil {
		return fmt.Errorf("invalid stored change request payload: %w", err)
	}
	if err := s.validateNoteChange(ctx, cr.RequesterID, cr.Action, p); err != nil {
		return err
	}
	switch cr.Action {
	case domain.ChangeNoteCreate:
		_, err := s.notes.Create(ctx, cr.RequesterID, p.OwnerType, p.OwnerID, p.Texto)
		return err
	case domain.ChangeNoteUpdate:
		_, err := s.notes.Update(ctx, *p.NoteID, cr.RequesterID, p.Texto)
		return err
	case domain.ChangeNoteDelete:
		return s.notes.Delete(ctx, *p.NoteID, cr.RequesterID)
	}
	return domain.ErrValidation
}
