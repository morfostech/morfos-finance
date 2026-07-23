// Command api is the Morfos Finance HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/morfostech/morfos-finance/internal/auth"
	"github.com/morfostech/morfos-finance/internal/config"
	"github.com/morfostech/morfos-finance/internal/database"
	apphttp "github.com/morfostech/morfos-finance/internal/http"
	"github.com/morfostech/morfos-finance/internal/http/handlers"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/migrate"
	"github.com/morfostech/morfos-finance/internal/repository"
	"github.com/morfostech/morfos-finance/internal/service"
	"github.com/morfostech/morfos-finance/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
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
	slog.Info("migrations applied")

	// Wiring: repositories -> services -> handlers.
	userRepo := repository.NewUserRepository(pool)
	projectRepo := repository.NewProjectRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	catRepo := repository.NewCategoryRepository(pool)
	recurrenceRepo := repository.NewRecurrenceRepository(pool)
	attachmentRepo := repository.NewAttachmentRepository(pool)

	store, _, err := storage.New(cfg.Storage)
	if err != nil {
		return err
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := service.NewAuthService(userRepo, tokens)
	projectSvc := service.NewProjectService(projectRepo)
	txSvc := service.NewTransactionService(txRepo)
	catSvc := service.NewCategoryService(catRepo)
	recurrenceSvc := service.NewRecurrenceService(recurrenceRepo)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, store, txRepo, projectRepo, projectRepo, cfg.Storage.MaxUploadBytes)
	dashboardSvc := service.NewDashboardService(repository.NewDashboardRepository(pool), recurrenceSvc, projectSvc)
	noteSvc := service.NewNoteService(repository.NewNoteRepository(pool))
	changeRequestSvc := service.NewChangeRequestService(repository.NewChangeRequestRepository(pool), noteSvc)
	planningSvc := service.NewPlanningService(repository.NewPlanningRepository(pool), recurrenceSvc)
	viaPermutaSvc := service.NewViaPermutaService(repository.NewViaPermutaRepository(pool))

	localUploadDir := ""
	if !cfg.Storage.UseS3() {
		localUploadDir = cfg.Storage.Dir
	}

	router := &apphttp.Router{
		Auth:           handlers.NewAuthHandler(authSvc),
		Projects:       handlers.NewProjectHandler(projectSvc),
		Transactions:   handlers.NewTransactionHandler(txSvc),
		Categories:     handlers.NewCategoryHandler(catSvc),
		Recurrence:     handlers.NewRecurrenceHandler(recurrenceSvc),
		Attachments:    handlers.NewAttachmentHandler(attachmentSvc, cfg.Storage.MaxUploadBytes),
		Dashboard:      handlers.NewDashboardHandler(dashboardSvc),
		Notes:          handlers.NewNoteHandler(noteSvc),
		ChangeRequests: handlers.NewChangeRequestHandler(changeRequestSvc),
		Planning:       handlers.NewPlanningHandler(planningSvc),
		ViaPermuta:     handlers.NewViaPermutaHandler(viaPermutaSvc),
		Authn:          middleware.NewAuthenticator(tokens),
		CORSOrigins:    cfg.CORSOrigins,
		LocalUploadDir: localUploadDir,
		FrontendDir:    cfg.FrontendDir,
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.Build(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
