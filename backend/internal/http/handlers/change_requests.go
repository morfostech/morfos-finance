package handlers

import (
	"net/http"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type ChangeRequestHandler struct {
	requests *service.ChangeRequestService
}

func NewChangeRequestHandler(requests *service.ChangeRequestService) *ChangeRequestHandler {
	return &ChangeRequestHandler{requests: requests}
}

type createChangeRequest struct {
	Action  domain.ChangeRequestAction `json:"action"`
	Payload service.NoteChangePayload  `json:"payload"`
}

type reviewChangeRequest struct {
	Comment string `json:"comment"`
}

func (h *ChangeRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createChangeRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	cr, err := h.requests.Create(r.Context(), service.Viewer{UserID: p.UserID, Role: p.Role}, req.Action, req.Payload)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, cr)
}

func (h *ChangeRequestHandler) List(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	list, err := h.requests.List(r.Context(), service.Viewer{UserID: p.UserID, Role: p.Role})
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if list == nil {
		list = []domain.ChangeRequest{}
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *ChangeRequestHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, true)
}

func (h *ChangeRequestHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, false)
}

func (h *ChangeRequestHandler) review(w http.ResponseWriter, r *http.Request, approve bool) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reviewChangeRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	viewer := service.Viewer{UserID: p.UserID, Role: p.Role}
	var err error
	if approve {
		err = h.requests.Approve(r.Context(), id, viewer, req.Comment)
	} else {
		err = h.requests.Reject(r.Context(), id, viewer, req.Comment)
	}
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}
