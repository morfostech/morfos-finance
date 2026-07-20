// Command seed provisions the initial admin user. Idempotent: skips if the email
// already exists. Categories are seeded via migration 0002; the admin lives here
// because its password hash must be generated at runtime, not baked into SQL.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/morfostech/morfos-finance/internal/config"
	"github.com/morfostech/morfos-finance/internal/database"
	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/migrate"
	"github.com/morfostech/morfos-finance/internal/repository"
	"github.com/morfostech/morfos-finance/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminSenha == "" {
		return errors.New("SEED_ADMIN_EMAIL e SEED_ADMIN_SENHA são obrigatórios")
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		return err
	}

	userRepo := repository.NewUserRepository(pool)
	// Token manager is unused for seeding; nil-safe since we only call CreateUser.
	svc := service.NewAuthService(userRepo, nil)

	if _, err := userRepo.GetByEmail(ctx, cfg.SeedAdminEmail); err == nil {
		slog.Info("admin já existe, nada a fazer", "email", cfg.SeedAdminEmail)
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	admin, err := svc.CreateUser(ctx, service.CreateUserInput{
		Nome:         cfg.SeedAdminNome,
		Email:        cfg.SeedAdminEmail,
		SenhaInicial: cfg.SeedAdminSenha,
		Role:         domain.RoleAdmin,
	})
	if err != nil {
		return err
	}
	slog.Info("admin criado", "id", admin.ID, "email", admin.Email,
		"must_change_password", admin.MustChangePassword)
	return nil
}
