package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/handlers"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
)

// Router wires middleware and routes. New feature modules register their routes
// here as they land.
type Router struct {
	Auth         *handlers.AuthHandler
	Projects     *handlers.ProjectHandler
	Transactions *handlers.TransactionHandler
	Categories   *handlers.CategoryHandler
	Recurrence   *handlers.RecurrenceHandler
	Authn        *middleware.Authenticator
	CORSOrigins  []string
}

func (rt *Router) Build() http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   rt.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", rt.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(rt.Authn.RequireAuth)
			r.Get("/auth/me", rt.Auth.Me)
			r.Post("/auth/change-password", rt.Auth.ChangePassword)

			// Projects: readable by any authenticated user (service scopes
			// colaboradores to their allocations); writes are admin-only.
			r.Get("/projects", rt.Projects.List)
			r.Get("/projects/{id}", rt.Projects.Get)

			// Transactions: reads scoped by role (colaborador -> own rows).
			r.Get("/transactions", rt.Transactions.List)
			r.Get("/transactions/{id}", rt.Transactions.Get)

			// Categories: readable by any authenticated user.
			r.Get("/categories", rt.Categories.List)

			// Company financial views: admin and sócio only.
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleSocio))
				r.Get("/recurrence", rt.Recurrence.Month)
				r.Get("/recurrence/timeline", rt.Recurrence.Timeline)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin))

				r.Post("/projects", rt.Projects.Create)
				r.Put("/projects/{id}", rt.Projects.Update)
				r.Put("/projects/{id}/members", rt.Projects.SetMembers)
				r.Patch("/projects/{id}/installments/{iid}", rt.Projects.MarkInstallment)

				r.Post("/transactions", rt.Transactions.Create)
				r.Put("/transactions/{id}", rt.Transactions.Update)
				r.Delete("/transactions/{id}", rt.Transactions.Delete)

				r.Post("/categories", rt.Categories.Create)
				r.Delete("/categories/{id}", rt.Categories.Delete)

				// User management.
				r.Get("/users", rt.Auth.ListUsers)
				r.Post("/users", rt.Auth.CreateUser)
				r.Post("/users/{id}/reset-password", rt.Auth.ResetPassword)
			})
		})
	})

	return r
}
