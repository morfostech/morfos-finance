package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// Claims is the JWT payload. Role and MustChangePassword travel in the token so
// most requests authorize without a DB round-trip; sensitive mutations re-check
// the DB.
type Claims struct {
	Role               domain.Role `json:"role"`
	MustChangePassword bool        `json:"mcp"`
	jwt.RegisteredClaims
}

// TokenManager issues and verifies HS256 tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, issuer: "morfos-finance"}
}

// Issue mints a signed token for the user. Subject is the user ID.
func (m *TokenManager) Issue(u *domain.User) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.ttl)
	claims := Claims{
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", u.ID),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("assinar token: %w", err)
	}
	return signed, exp, nil
}

// Parse verifies the signature and expiry and returns the claims.
func (m *TokenManager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo inesperado: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
