// Package config loads runtime configuration from the environment (.env in dev).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	CORSOrigins []string

	// Seed admin (used by cmd/seed).
	SeedAdminNome  string
	SeedAdminEmail string
	SeedAdminSenha string
}

// Load reads .env if present (ignored in prod where vars are injected), then
// pulls config from the environment, failing fast on missing essentials.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTTTL:         getdur("JWT_TTL", 24*time.Hour),
		CORSOrigins:    splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		SeedAdminNome:  getenv("SEED_ADMIN_NOME", "Admin"),
		SeedAdminEmail: os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminSenha: os.Getenv("SEED_ADMIN_SENHA"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL é obrigatório")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET é obrigatório")
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
