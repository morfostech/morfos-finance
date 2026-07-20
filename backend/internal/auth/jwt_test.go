package auth

import (
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

func TestTokenIssueParse(t *testing.T) {
	tm := NewTokenManager("segredo-de-teste", time.Hour)
	u := &domain.User{ID: 42, Role: domain.RoleAdmin, MustChangePassword: true}

	token, exp, err := tm.Issue(u)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !exp.After(time.Now()) {
		t.Fatal("expiry not in the future")
	}

	claims, err := tm.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != "42" {
		t.Errorf("subject = %q, want 42", claims.Subject)
	}
	if claims.Role != domain.RoleAdmin {
		t.Errorf("role = %q, want admin", claims.Role)
	}
	if !claims.MustChangePassword {
		t.Error("must_change_password lost in round-trip")
	}
}

func TestTokenExpired(t *testing.T) {
	tm := NewTokenManager("segredo", -time.Minute) // already expired
	token, _, _ := tm.Issue(&domain.User{ID: 1, Role: domain.RoleColaborador})
	if _, err := tm.Parse(token); err == nil {
		t.Fatal("expected expired token to fail parsing")
	}
}

func TestTokenWrongSecret(t *testing.T) {
	issuer := NewTokenManager("segredo-a", time.Hour)
	token, _, _ := issuer.Issue(&domain.User{ID: 1, Role: domain.RoleSocio})

	verifier := NewTokenManager("segredo-b", time.Hour)
	if _, err := verifier.Parse(token); err == nil {
		t.Fatal("expected signature mismatch to fail")
	}
}
