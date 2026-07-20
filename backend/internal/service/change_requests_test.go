package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type fakeChangeRequestRepo struct {
	requests map[int64]*domain.ChangeRequest
	nextID   int64
}

func newFakeChangeRequestRepo() *fakeChangeRequestRepo {
	return &fakeChangeRequestRepo{requests: make(map[int64]*domain.ChangeRequest), nextID: 1}
}

func (f *fakeChangeRequestRepo) Create(_ context.Context, cr *domain.ChangeRequest) (*domain.ChangeRequest, error) {
	cr.ID = f.nextID
	f.nextID++
	cp := *cr
	f.requests[cr.ID] = &cp
	return cr, nil
}

func (f *fakeChangeRequestRepo) List(_ context.Context, requesterID *int64) ([]domain.ChangeRequest, error) {
	var out []domain.ChangeRequest
	for _, cr := range f.requests {
		if requesterID == nil || cr.RequesterID == *requesterID {
			out = append(out, *cr)
		}
	}
	return out, nil
}

func (f *fakeChangeRequestRepo) StartReview(_ context.Context, id int64) (*domain.ChangeRequest, error) {
	cr, ok := f.requests[id]
	if !ok || cr.Status != domain.ChangePending {
		return nil, domain.ErrConflict
	}
	cr.Status = domain.ChangeProcessing
	return cr, nil
}

func (f *fakeChangeRequestRepo) FinishReview(_ context.Context, id, reviewerID int64, status domain.ChangeRequestStatus, comment *string) error {
	cr := f.requests[id]
	if cr == nil || cr.Status != domain.ChangeProcessing {
		return domain.ErrConflict
	}
	cr.Status = status
	cr.ReviewerID = &reviewerID
	cr.ReviewComment = comment
	return nil
}

func (f *fakeChangeRequestRepo) RestorePending(_ context.Context, id int64) error {
	if cr := f.requests[id]; cr != nil {
		cr.Status = domain.ChangePending
	}
	return nil
}

func TestCollaboratorNoteChangeRequiresApproval(t *testing.T) {
	ctx := context.Background()
	noteRepo := newFakeNoteRepo()
	requestRepo := newFakeChangeRequestRepo()
	svc := NewChangeRequestService(requestRepo, NewNoteService(noteRepo))
	viewer := Viewer{UserID: 10, Role: domain.RoleColaborador}

	request, err := svc.Create(ctx, viewer, domain.ChangeNoteCreate, NoteChangePayload{
		OwnerType: domain.NoteOwnerGeral,
		Texto:     "Aguardando aprovação",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(noteRepo.store) != 0 {
		t.Fatal("note was created before approval")
	}
	if request.Status != domain.ChangePending {
		t.Fatalf("status = %q, want pending", request.Status)
	}

	err = svc.Approve(ctx, request.ID, Viewer{UserID: 1, Role: domain.RoleAdmin}, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if len(noteRepo.store) != 1 {
		t.Fatalf("notes = %d, want 1 after approval", len(noteRepo.store))
	}
	if requestRepo.requests[request.ID].Status != domain.ChangeApproved {
		t.Fatalf("status = %q, want approved", requestRepo.requests[request.ID].Status)
	}
}

func TestChangeRequestRoleAndRejectionRules(t *testing.T) {
	svc := NewChangeRequestService(newFakeChangeRequestRepo(), NewNoteService(newFakeNoteRepo()))
	ctx := context.Background()

	_, err := svc.Create(ctx, Viewer{UserID: 1, Role: domain.RoleAdmin}, domain.ChangeNoteCreate, NoteChangePayload{
		OwnerType: domain.NoteOwnerGeral,
		Texto:     "direct write instead",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("admin create err = %v, want validation", err)
	}

	req, err := svc.Create(ctx, Viewer{UserID: 2, Role: domain.RoleColaborador}, domain.ChangeNoteCreate, NoteChangePayload{
		OwnerType: domain.NoteOwnerGeral,
		Texto:     "request",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Reject(ctx, req.ID, Viewer{UserID: 1, Role: domain.RoleSocio}, " "); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("blank rejection err = %v, want validation", err)
	}
	if err := svc.Reject(ctx, req.ID, Viewer{UserID: 1, Role: domain.RoleSocio}, "Não aplicável"); err != nil {
		t.Fatal(err)
	}
}
