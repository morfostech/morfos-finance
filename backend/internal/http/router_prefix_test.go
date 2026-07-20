package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFinancePrefixRouting(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	})
	handler := withFinancePrefix(inner)

	prefixed := httptest.NewRecorder()
	handler.ServeHTTP(prefixed, httptest.NewRequest(http.MethodGet, "/finance/api/projects?active=true", nil))
	if got := prefixed.Body.String(); got != "/api/projects" {
		t.Fatalf("prefixed path = %q, want /api/projects", got)
	}

	direct := httptest.NewRecorder()
	handler.ServeHTTP(direct, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := direct.Body.String(); got != "/health" {
		t.Fatalf("direct path = %q, want /health", got)
	}

	redirect := httptest.NewRecorder()
	handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/finance", nil))
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/finance/" {
		t.Fatalf("redirect = %d %q", redirect.Code, redirect.Header().Get("Location"))
	}
}
