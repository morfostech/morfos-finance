package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/auth"
	"github.com/morfostech/morfos-finance/internal/domain"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuth(t *testing.T) {
	tm := auth.NewTokenManager("segredo", time.Hour)
	authn := NewAuthenticator(tm)
	token, _, _ := tm.Issue(&domain.User{ID: 7, Role: domain.RoleColaborador})

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"sem header", "", http.StatusUnauthorized},
		{"token inválido", "Bearer lixo", http.StatusUnauthorized},
		{"token válido", "Bearer " + token, http.StatusOK},
		{"case-insensitive bearer", "bearer " + token, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()
			authn.RequireAuth(okHandler()).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tm := auth.NewTokenManager("segredo", time.Hour)
	authn := NewAuthenticator(tm)

	tests := []struct {
		name       string
		role       domain.Role
		allow      []domain.Role
		wantStatus int
	}{
		{"admin acessa área admin", domain.RoleAdmin, []domain.Role{domain.RoleAdmin}, http.StatusOK},
		{"colaborador barrado em área admin", domain.RoleColaborador, []domain.Role{domain.RoleAdmin}, http.StatusForbidden},
		{"socio permitido em leitura", domain.RoleSocio, []domain.Role{domain.RoleAdmin, domain.RoleSocio}, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, _, _ := tm.Issue(&domain.User{ID: 1, Role: tc.role})
			// Chain: RequireAuth -> RequireRole -> handler.
			h := authn.RequireAuth(RequireRole(tc.allow...)(okHandler()))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}
