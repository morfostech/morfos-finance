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

	Storage StorageConfig

	// Seed admin (used by cmd/seed).
	SeedAdminNome  string
	SeedAdminEmail string
	SeedAdminSenha string
}

// StorageConfig selects and configures object storage. When Endpoint and Bucket
// are set, the S3-compatible backend (Cloudflare R2 by default) is used;
// otherwise files go to local disk under Dir.
type StorageConfig struct {
	// S3-compatible.
	Endpoint      string
	Bucket        string
	AccessKey     string
	SecretKey     string
	Region        string
	PublicBaseURL string

	// Local disk fallback.
	Dir string

	MaxUploadBytes int64
}

// UseS3 reports whether the S3-compatible backend is configured.
func (s StorageConfig) UseS3() bool {
	return s.Endpoint != "" && s.Bucket != ""
}

// Load reads .env if present (ignored in prod where vars are injected), then
// pulls config from the environment, failing fast on missing essentials.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      getdur("JWT_TTL", 24*time.Hour),
		CORSOrigins: splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		Storage: StorageConfig{
			Endpoint:       os.Getenv("S3_ENDPOINT"),
			Bucket:         os.Getenv("S3_BUCKET"),
			AccessKey:      os.Getenv("S3_ACCESS_KEY_ID"),
			SecretKey:      os.Getenv("S3_SECRET_ACCESS_KEY"),
			Region:         getenv("S3_REGION", "auto"),
			PublicBaseURL:  os.Getenv("S3_PUBLIC_BASE_URL"),
			Dir:            getenv("UPLOAD_DIR", "./uploads"),
			MaxUploadBytes: int64(getint("MAX_UPLOAD_MB", 10)) * 1024 * 1024,
		},
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

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
