package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type ProjectHandler struct {
	projects *service.ProjectService
}

func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// projectRequest is the create/update payload. Monetary fields are in centavos.
type projectRequest struct {
	Nome               string       `json:"nome"`
	Cliente            *string      `json:"cliente"`
	ValorImplementacao *int64       `json:"valor_implementacao"`
	ValorMensal        *int64       `json:"valor_mensal"`
	DiaVencimento      *int         `json:"dia_vencimento"`
	DataInicio         *domain.Date `json:"data_inicio"`
	DataFim            *domain.Date `json:"data_fim"`
	Status             string       `json:"status"`
	MemberIDs          []int64      `json:"member_ids"`
}

func (r projectRequest) toInput() service.ProjectInput {
	return service.ProjectInput{
		Nome:               r.Nome,
		Cliente:            r.Cliente,
		ValorImplementacao: moneyPtr(r.ValorImplementacao),
		ValorMensal:        moneyPtr(r.ValorMensal),
		DiaVencimento:      r.DiaVencimento,
		DataInicio:         r.DataInicio,
		DataFim:            r.DataFim,
		Status:             domain.ProjectStatus(r.Status),
		MemberIDs:          r.MemberIDs,
	}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req projectRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.projects.Create(r.Context(), req.toInput())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, p)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req projectRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.projects.Update(r.Context(), id, req.toInput())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := h.projects.Get(r.Context(), id, viewerFrom(r))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context(), viewerFrom(r))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	respond.JSON(w, http.StatusOK, projects)
}

type membersRequest struct {
	MemberIDs []int64 `json:"member_ids"`
}

func (h *ProjectHandler) SetMembers(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req membersRequest
	if !decode(w, r, &req) {
		return
	}
	if err := h.projects.SetMembers(r.Context(), id, req.MemberIDs); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

// installmentRequest marks an installment paid (pago_em set) or pending (null).
type installmentRequest struct {
	PagoEm *domain.Date `json:"pago_em"`
}

func (h *ProjectHandler) MarkInstallment(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(w, r)
	if !ok {
		return
	}
	installmentID, err := strconv.ParseInt(chi.URLParam(r, "iid"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "id da parcela inválido")
		return
	}
	var req installmentRequest
	if !decode(w, r, &req) {
		return
	}
	inst, err := h.projects.MarkInstallment(r.Context(), projectID, installmentID, req.PagoEm)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, inst)
}

// --- helpers ---

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return id, true
}

func viewerFrom(r *http.Request) service.Viewer {
	p, _ := middleware.PrincipalFrom(r.Context())
	return service.Viewer{UserID: p.UserID, Role: p.Role}
}

func moneyPtr(cents *int64) *domain.Money {
	if cents == nil {
		return nil
	}
	m := domain.Money(*cents)
	return &m
}
