package handlers

import (
	"net/http"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type NoteHandler struct {
	notes *service.NoteService
}

func NewNoteHandler(notes *service.NoteService) *NoteHandler {
	return &NoteHandler{notes: notes}
}

type noteRequest struct {
	OwnerType string `json:"owner_type"`
	OwnerID   *int64 `json:"owner_id"`
	Texto     string `json:"texto"`
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req noteRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	n, err := h.notes.Create(r.Context(), p.UserID, domain.NoteOwner(req.OwnerType), req.OwnerID, req.Texto)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, n)
}

// List handles GET /api/notes?owner_type=&owner_id= — always scoped to the
// caller; there is no cross-user listing.
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerType := domain.NoteOwner(r.URL.Query().Get("owner_type"))
	ownerID, err := queryInt(r.URL.Query().Get("owner_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "owner_id inválido")
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	notes, err := h.notes.List(r.Context(), p.UserID, ownerType, ownerID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if notes == nil {
		notes = []domain.Note{}
	}
	respond.JSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req noteRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	n, err := h.notes.Update(r.Context(), id, p.UserID, req.Texto)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, n)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	if err := h.notes.Delete(r.Context(), id, p.UserID); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}
