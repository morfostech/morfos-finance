// Package middleware provides authentication and role-based authorization.
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/morfostech/morfos-finance/internal/auth"
	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/respond"
)

type ctxKey int

const principalKey ctxKey = iota

// Principal is the authenticated identity extracted from the JWT.
type Principal struct {
	UserID             int64
	Role               domain.Role
	MustChangePassword bool
}

// Authenticator builds middleware that validates the Bearer token and injects
// the Principal into the request context.
type Authenticator struct {
	tokens *auth.TokenManager
}

func NewAuthenticator(tokens *auth.TokenManager) *Authenticator {
	return &Authenticator{tokens: tokens}
}

// RequireAuth rejects requests without a valid Bearer token.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			respond.Error(w, http.StatusUnauthorized, "token ausente")
			return
		}
		claims, err := a.tokens.Parse(raw)
		if err != nil {
			respond.Error(w, http.StatusUnauthorized, "token inválido ou expirado")
			return
		}
		id, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			respond.Error(w, http.StatusUnauthorized, "token inválido")
			return
		}
		p := Principal{UserID: id, Role: claims.Role, MustChangePassword: claims.MustChangePassword}
		ctx := context.WithValue(r.Context(), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole allows the request only if the principal holds one of the roles.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := make(map[domain.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				respond.Error(w, http.StatusUnauthorized, "não autenticado")
				return
			}
			if !allowed[p.Role] {
				respond.Error(w, http.StatusForbidden, "acesso negado")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PrincipalFrom extracts the authenticated principal from the context.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
