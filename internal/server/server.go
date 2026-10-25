// Package server wires repositories, services and controllers together and
// runs the HTTP server.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/config"
	"github.com/maulanaiskak/portfolio-api/internal/controller"
	"github.com/maulanaiskak/portfolio-api/internal/middleware"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type Services struct {
	Cv        service.CvService
	AdminUser service.AdminUserService
}

// NewServices builds the service graph on top of the Postgres repositories.
// Shared by the HTTP server and CLI commands (e.g. cmd/seedcv).
func NewServices(pool *pgxpool.Pool) Services {
	return Services{
		Cv: service.NewCvService(
			repository.NewCvExperienceRepository(pool),
			repository.NewCvProjectRepository(pool),
			repository.NewCvEducationRepository(pool),
			repository.NewCvSkillRepository(pool),
			repository.NewTransactor(pool),
		),
		AdminUser: service.NewAdminUserService(repository.NewAdminUserRepository(pool)),
	}
}

func NewRouter(cfg config.Config, pool *pgxpool.Pool, svc Services) http.Handler {
	r := chi.NewRouter()

	if cfg.TrustProxy {
		r.Use(chimw.RealIP)
	}
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	r.Use(middleware.Cors(cfg.AllowedOrigins))
	r.Use(middleware.BodyLimit(cfg.MaxBodyBytes))
	r.Use(chimw.Timeout(cfg.WriteTimeout - time.Second))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { response.Error(w, r, response.ErrRouteNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { response.Error(w, r, response.ErrMethodNotAllowed) })

	controller.NewHealthController(r, pool)

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Group(func(public chi.Router) {
			public.Use(middleware.RateLimitByIP(cfg.RateLimitPerMinute))
			controller.NewCvController(public, svc.Cv)
		})
		// Phase 2: v1.Route("/admin", ...) behind JWT auth middleware.
	})

	return r
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func Run(ctx context.Context, cfg config.Config, handler http.Handler) error {
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
