package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type RecurrenceHandler struct {
	recurrence *service.RecurrenceService
}

func NewRecurrenceHandler(recurrence *service.RecurrenceService) *RecurrenceHandler {
	return &RecurrenceHandler{recurrence: recurrence}
}

// Month handles GET /api/recurrence?ano=&mes=&project_id=.
// ano/mes default to the current month.
func (h *RecurrenceHandler) Month(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	ano := queryIntDefault(r, "ano", now.Year())
	mes := queryIntDefault(r, "mes", int(now.Month()))
	projectID, err := queryInt(r.URL.Query().Get("project_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "project_id inválido")
		return
	}
	summary, err := h.recurrence.Month(r.Context(), ano, mes, projectID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, summary)
}

// Timeline handles GET /api/recurrence/timeline?ano=&project_id=, returning the
// 12 monthly summaries for the year.
func (h *RecurrenceHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	ano := queryIntDefault(r, "ano", time.Now().Year())
	projectID, err := queryInt(r.URL.Query().Get("project_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "project_id inválido")
		return
	}
	timeline, err := h.recurrence.Year(r.Context(), ano, projectID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, timeline)
}

func queryIntDefault(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
