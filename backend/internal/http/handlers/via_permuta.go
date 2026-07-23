package handlers

import (
	"net/http"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type ViaPermutaHandler struct {
	service *service.ViaPermutaService
}

func NewViaPermutaHandler(service *service.ViaPermutaService) *ViaPermutaHandler {
	return &ViaPermutaHandler{service: service}
}

type vpTransactionRequest struct {
	Tipo        string       `json:"tipo"`
	Status      string       `json:"status"`
	Valor       int64        `json:"valor"`
	Data        *domain.Date `json:"data"`
	Permutante  string       `json:"permutante"`
	Oferta      string       `json:"oferta"`
	ProjectID   *int64       `json:"project_id"`
	VoucherCode *string      `json:"voucher_code"`
	Observacoes *string      `json:"observacoes"`
}

func (r vpTransactionRequest) input() service.VPTransactionInput {
	return service.VPTransactionInput{
		Tipo: domain.VPTransactionType(r.Tipo), Status: domain.VPTransactionStatus(r.Status),
		Valor: domain.Money(r.Valor), Data: r.Data, Permutante: r.Permutante,
		Oferta: r.Oferta, ProjectID: r.ProjectID, VoucherCode: r.VoucherCode,
		Observacoes: r.Observacoes,
	}
}

func (h *ViaPermutaHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req vpTransactionRequest
	if !decode(w, r, &req) {
		return
	}
	principal, _ := middleware.PrincipalFrom(r.Context())
	t, err := h.service.CreateTransaction(r.Context(), req.input(), principal.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, t)
}

func (h *ViaPermutaHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req vpTransactionRequest
	if !decode(w, r, &req) {
		return
	}
	t, err := h.service.UpdateTransaction(r.Context(), id, req.input())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, t)
}

func (h *ViaPermutaHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteTransaction(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

func vpFilterFromRequest(r *http.Request) (domain.VPTransactionFilter, error) {
	q := r.URL.Query()
	var filter domain.VPTransactionFilter
	if value := q.Get("from"); value != "" {
		date, err := parseDate(value)
		if err != nil {
			return filter, err
		}
		filter.From = date
	}
	if value := q.Get("to"); value != "" {
		date, err := parseDate(value)
		if err != nil {
			return filter, err
		}
		filter.To = date
	}
	if value := q.Get("tipo"); value != "" {
		tipo := domain.VPTransactionType(value)
		filter.Tipo = &tipo
	}
	if value := q.Get("status"); value != "" {
		status := domain.VPTransactionStatus(value)
		filter.Status = &status
	}
	return filter, nil
}

func (h *ViaPermutaHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	filter, err := vpFilterFromRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.service.ListTransactions(r.Context(), filter)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if items == nil {
		items = []domain.VPTransaction{}
	}
	respond.JSON(w, http.StatusOK, items)
}

func (h *ViaPermutaHandler) Summary(w http.ResponseWriter, r *http.Request) {
	filter, err := vpFilterFromRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	summary, err := h.service.Summary(r.Context(), domain.VPTransactionFilter{From: filter.From, To: filter.To})
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, summary)
}

func (h *ViaPermutaHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetSettings(r.Context())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, settings)
}

func (h *ViaPermutaHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CreditLimit int64 `json:"limite_credito"`
	}
	if !decode(w, r, &req) {
		return
	}
	settings, err := h.service.UpdateSettings(r.Context(), domain.Money(req.CreditLimit))
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, settings)
}

type vpOfferRequest struct {
	Titulo      string  `json:"titulo"`
	Descricao   *string `json:"descricao"`
	Valor       *int64  `json:"valor"`
	Negociavel  bool    `json:"negociavel"`
	Status      string  `json:"status"`
	ExternalURL *string `json:"external_url"`
}

func (r vpOfferRequest) input() service.VPOfferInput {
	var amount *domain.Money
	if r.Valor != nil {
		value := domain.Money(*r.Valor)
		amount = &value
	}
	return service.VPOfferInput{
		Titulo: r.Titulo, Descricao: r.Descricao, Valor: amount,
		Negociavel: r.Negociavel, Status: domain.VPOfferStatus(r.Status),
		ExternalURL: r.ExternalURL,
	}
}

func (h *ViaPermutaHandler) ListOffers(w http.ResponseWriter, r *http.Request) {
	offers, err := h.service.ListOffers(r.Context())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if offers == nil {
		offers = []domain.VPOffer{}
	}
	respond.JSON(w, http.StatusOK, offers)
}

func (h *ViaPermutaHandler) CreateOffer(w http.ResponseWriter, r *http.Request) {
	var req vpOfferRequest
	if !decode(w, r, &req) {
		return
	}
	principal, _ := middleware.PrincipalFrom(r.Context())
	offer, err := h.service.CreateOffer(r.Context(), req.input(), principal.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, offer)
}

func (h *ViaPermutaHandler) UpdateOffer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req vpOfferRequest
	if !decode(w, r, &req) {
		return
	}
	offer, err := h.service.UpdateOffer(r.Context(), id, req.input())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, offer)
}

func (h *ViaPermutaHandler) DeleteOffer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteOffer(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}
