package handlers

import (
	"net/http"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type CategoryHandler struct {
	cats *service.CategoryService
}

func NewCategoryHandler(cats *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{cats: cats}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	cats, err := h.cats.List(r.Context())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if cats == nil {
		cats = []domain.ExpenseCategory{}
	}
	respond.JSON(w, http.StatusOK, cats)
}

type categoryRequest struct {
	Nome string `json:"nome"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if !decode(w, r, &req) {
		return
	}
	c, err := h.cats.Create(r.Context(), req.Nome)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, c)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.cats.Delete(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}
