package handlers

import (
	"net/http"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type DashboardHandler struct {
	dashboard *service.DashboardService
}

func NewDashboardHandler(dashboard *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// Company handles GET /api/dashboard/company?from=&to=. Defaults to the current month.
func (h *DashboardHandler) Company(w http.ResponseWriter, r *http.Request) {
	from, to, err := periodOrCurrentMonth(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	dash, err := h.dashboard.Company(r.Context(), from, to)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, dash)
}

// Me handles GET /api/dashboard/me?from=&to=. Defaults to the current month.
func (h *DashboardHandler) Me(w http.ResponseWriter, r *http.Request) {
	from, to, err := periodOrCurrentMonth(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	dash, err := h.dashboard.Me(r.Context(), viewerFrom(r), from, to)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, dash)
}

// periodOrCurrentMonth reads from/to (YYYY-MM-DD), defaulting to the current
// calendar month. Supports the diário/semanal/mensal/anual/personalizado filters
// entirely via an explicit [from, to] chosen by the client.
func periodOrCurrentMonth(r *http.Request) (from, to domain.Date, err error) {
	q := r.URL.Query()
	fromStr, toStr := q.Get("from"), q.Get("to")

	if fromStr == "" && toStr == "" {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return domain.NewDate(start), domain.NewDate(start.AddDate(0, 1, -1)), nil
	}

	fp, err := parseDate(fromStr)
	if err != nil {
		return from, to, err
	}
	tp, err := parseDate(toStr)
	if err != nil {
		return from, to, err
	}
	if fp == nil || tp == nil {
		return from, to, errBadPeriod
	}
	if tp.Time.Before(fp.Time) {
		return from, to, errBadPeriod
	}
	return *fp, *tp, nil
}

var errBadPeriod = &periodError{}

type periodError struct{}

func (*periodError) Error() string { return "período inválido: informe from e to (YYYY-MM-DD)" }
