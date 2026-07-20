// Package handlers contains the HTTP transport layer: decode, delegate to
// services, encode. No business rules live here.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type loginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *domain.User `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decode(w, r, &req) {
		return
	}
	res, err := h.auth.Login(r.Context(), req.Email, req.Senha)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, loginResponse{
		Token: res.Token, ExpiresAt: res.ExpiresAt, User: res.User,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	u, err := h.auth.Me(r.Context(), p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, u)
}

type changePasswordRequest struct {
	SenhaAtual string `json:"senha_atual"`
	NovaSenha  string `json:"nova_senha"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	var req changePasswordRequest
	if !decode(w, r, &req) {
		return
	}
	if err := h.auth.ChangePassword(r.Context(), p.UserID, req.SenhaAtual, req.NovaSenha); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

// --- admin: user management ---

type createUserRequest struct {
	Nome         string `json:"nome"`
	Email        string `json:"email"`
	SenhaInicial string `json:"senha_inicial"`
	Role         string `json:"role"`
}

func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decode(w, r, &req) {
		return
	}
	u, err := h.auth.CreateUser(r.Context(), service.CreateUserInput{
		Nome:         req.Nome,
		Email:        req.Email,
		SenhaInicial: req.SenhaInicial,
		Role:         domain.Role(req.Role),
	})
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, u)
}

func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if users == nil {
		users = []domain.User{}
	}
	respond.JSON(w, http.StatusOK, users)
}

type resetPasswordRequest struct {
	NovaSenha string `json:"nova_senha"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req resetPasswordRequest
	if !decode(w, r, &req) {
		return
	}
	if err := h.auth.ResetPassword(r.Context(), id, req.NovaSenha); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

// decode reads a JSON body, rejecting unknown fields. Returns false (and writes
// a 400) on failure.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respond.Error(w, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}
