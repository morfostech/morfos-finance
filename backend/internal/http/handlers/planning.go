package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type PlanningHandler struct{ planning *service.PlanningService }

func NewPlanningHandler(planning *service.PlanningService) *PlanningHandler {
	return &PlanningHandler{planning: planning}
}

type plannedRequest struct {
	Tipo         string       `json:"tipo"`
	Valor        int64        `json:"valor"`
	DueDate      *domain.Date `json:"due_date"`
	ProjectID    *int64       `json:"project_id"`
	UserID       *int64       `json:"user_id"`
	Origem       *string      `json:"origem"`
	CategoryID   *int64       `json:"category_id"`
	Descricao    string       `json:"descricao"`
	RepeatMonths int          `json:"repeat_months"`
}

func (r plannedRequest) input() service.PlannedInput {
	var origem *domain.TxOrigem
	if r.Origem != nil {
		value := domain.TxOrigem(*r.Origem)
		origem = &value
	}
	return service.PlannedInput{Tipo: domain.TxType(r.Tipo), Valor: domain.Money(r.Valor),
		DueDate: r.DueDate, ProjectID: r.ProjectID, UserID: r.UserID, Origem: origem,
		CategoryID: r.CategoryID, Descricao: r.Descricao, RepeatMonths: r.RepeatMonths}
}

func (h *PlanningHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req plannedRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	items, err := h.planning.Create(r.Context(), req.input(), p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, items)
}

func (h *PlanningHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req plannedRequest
	if !decode(w, r, &req) {
		return
	}
	item, err := h.planning.Update(r.Context(), id, req.input())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, item)
}

func (h *PlanningHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.planning.Delete(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

func (h *PlanningHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f domain.PlanningFilter
	var err error
	if q.Get("from") != "" {
		f.From, err = parseDate(q.Get("from"))
		if err != nil {
			respond.Error(w, 400, err.Error())
			return
		}
	}
	if q.Get("to") != "" {
		f.To, err = parseDate(q.Get("to"))
		if err != nil {
			respond.Error(w, 400, err.Error())
			return
		}
	}
	if value := q.Get("status"); value != "" {
		status := domain.PlannedStatus(value)
		f.Status = &status
	}
	location, locationErr := time.LoadLocation("America/Sao_Paulo")
	if locationErr != nil {
		location = time.FixedZone("BRT", -3*60*60)
	}
	items, err := h.planning.List(r.Context(), f, time.Now().In(location))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, items)
}

func (h *PlanningHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Data *domain.Date `json:"data"`
	}
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	item, err := h.planning.Complete(r.Context(), id, p.UserID, req.Data)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, item)
}

func (h *PlanningHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	from := domain.NewDate(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
	to := domain.NewDate(from.AddDate(0, 3, 0))
	var err error
	if value := r.URL.Query().Get("from"); value != "" {
		parsed, e := parseDate(value)
		err = e
		if parsed != nil {
			from = *parsed
		}
	}
	if err == nil {
		if value := r.URL.Query().Get("to"); value != "" {
			parsed, e := parseDate(value)
			err = e
			if parsed != nil {
				to = *parsed
			}
		}
	}
	if err != nil {
		respond.Error(w, 400, err.Error())
		return
	}
	forecast, err := h.planning.Forecast(r.Context(), from, to)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, forecast)
}

type budgetRequest struct {
	CategoryID int64 `json:"category_id"`
	Ano        int   `json:"ano"`
	Mes        int   `json:"mes"`
	Valor      int64 `json:"valor"`
}

func (h *PlanningHandler) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	var req budgetRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	item, err := h.planning.UpsertBudget(r.Context(), req.CategoryID, req.Ano, req.Mes, domain.Money(req.Valor), p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, item)
}

func (h *PlanningHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	year, month := now.Year(), int(now.Month())
	var err error
	if value := r.URL.Query().Get("ano"); value != "" {
		year, err = strconv.Atoi(value)
	}
	if err == nil {
		if value := r.URL.Query().Get("mes"); value != "" {
			month, err = strconv.Atoi(value)
		}
	}
	if err != nil {
		respond.Error(w, 400, "mês inválido")
		return
	}
	items, err := h.planning.ListBudgets(r.Context(), year, month)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, items)
}

func (h *PlanningHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.planning.DeleteBudget(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}
