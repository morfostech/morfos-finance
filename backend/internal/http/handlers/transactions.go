package handlers

import (
	"net/http"
	"strconv"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type TransactionHandler struct {
	txs *service.TransactionService
}

func NewTransactionHandler(txs *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txs: txs}
}

// transactionRequest is the create/update payload. Valor is in centavos.
type transactionRequest struct {
	Tipo       string       `json:"tipo"`
	Valor      int64        `json:"valor"`
	Data       *domain.Date `json:"data"`
	ProjectID  *int64       `json:"project_id"`
	UserID     *int64       `json:"user_id"`
	Origem     *string      `json:"origem"`
	CategoryID *int64       `json:"category_id"`
	Descricao  *string      `json:"descricao"`
}

func (r transactionRequest) toInput() service.TransactionInput {
	var origem *domain.TxOrigem
	if r.Origem != nil {
		o := domain.TxOrigem(*r.Origem)
		origem = &o
	}
	return service.TransactionInput{
		Tipo:       domain.TxType(r.Tipo),
		Valor:      domain.Money(r.Valor),
		Data:       r.Data,
		ProjectID:  r.ProjectID,
		UserID:     r.UserID,
		Origem:     origem,
		CategoryID: r.CategoryID,
		Descricao:  r.Descricao,
	}
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req transactionRequest
	if !decode(w, r, &req) {
		return
	}
	p, _ := middleware.PrincipalFrom(r.Context())
	t, err := h.txs.Create(r.Context(), req.toInput(), p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, t)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req transactionRequest
	if !decode(w, r, &req) {
		return
	}
	t, err := h.txs.Update(r.Context(), id, req.toInput())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, t)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.txs.Delete(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := h.txs.Get(r.Context(), id, viewerFrom(r))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, t)
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTransactionFilter(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	txs, err := h.txs.List(r.Context(), filter, viewerFrom(r))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if txs == nil {
		txs = []domain.Transaction{}
	}
	respond.JSON(w, http.StatusOK, txs)
}

// parseTransactionFilter reads the query string into a domain filter.
// Supported: from, to (YYYY-MM-DD), tipo, origem, project_id, user_id, category_id.
func parseTransactionFilter(r *http.Request) (domain.TransactionFilter, error) {
	q := r.URL.Query()
	var f domain.TransactionFilter

	if v := q.Get("from"); v != "" {
		d, err := parseDate(v)
		if err != nil {
			return f, err
		}
		f.From = d
	}
	if v := q.Get("to"); v != "" {
		d, err := parseDate(v)
		if err != nil {
			return f, err
		}
		f.To = d
	}
	if v := q.Get("tipo"); v != "" {
		t := domain.TxType(v)
		f.Tipo = &t
	}
	if v := q.Get("origem"); v != "" {
		o := domain.TxOrigem(v)
		f.Origem = &o
	}
	var err error
	if f.ProjectID, err = queryInt(q.Get("project_id")); err != nil {
		return f, err
	}
	if f.UserID, err = queryInt(q.Get("user_id")); err != nil {
		return f, err
	}
	if f.CategoryID, err = queryInt(q.Get("category_id")); err != nil {
		return f, err
	}
	return f, nil
}

func parseDate(s string) (*domain.Date, error) {
	var d domain.Date
	if err := d.UnmarshalJSON([]byte(`"` + s + `"`)); err != nil {
		return nil, err
	}
	return &d, nil
}

func queryInt(s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
