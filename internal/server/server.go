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
	"github.com/maulanaiskak/portfolio-api/internal/security"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type Services struct {
	Cv        service.CvService
	AdminUser service.AdminUserService
	Auth      service.AuthService
	Trip      service.TripService
	Milestone service.MilestoneService
	Post      service.PostService
	Upload    service.UploadService
}

// NewServices builds the service graph on top of the Postgres repositories
// and the given object storage. Shared by the HTTP server and CLI commands.
func NewServices(cfg config.Config, pool *pgxpool.Pool, storage repository.ObjectStorage) Services {
	tx := repository.NewTransactor(pool)
	cache := repository.NewWebCache(cfg.WebRevalidateURL, cfg.RevalidateSecret)
	adminRepo := repository.NewAdminUserRepository(pool)
	return Services{
		Cv: service.NewCvService(
			repository.NewCvExperienceRepository(pool),
			repository.NewCvProjectRepository(pool),
			repository.NewCvEducationRepository(pool),
			repository.NewCvSkillRepository(pool),
			tx,
			cache,
		),
		AdminUser: service.NewAdminUserService(adminRepo, tx),
		Auth:      service.NewAuthService(adminRepo, security.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)),
		Trip:      service.NewTripService(repository.NewTripRepository(pool), tx, cache),
		Milestone: service.NewMilestoneService(repository.NewMilestoneRepository(pool), tx, cache),
		Post:      service.NewPostService(repository.NewPostRepository(pool), tx, cache),
		Upload:    service.NewUploadService(storage, cfg.MaxUploadBytes),
	}
}

// NewStorage returns the configured object storage. For the local driver it
// also returns the handler that serves the files.
func NewStorage(cfg config.Config) (repository.ObjectStorage, http.Handler, error) {
	if cfg.Driver == "r2" {
		return repository.NewR2Storage(cfg.R2Endpoint, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBaseURL), nil, nil
	}
	local, err := repository.NewLocalStorage(cfg.UploadDir, cfg.PublicBaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("local storage: %w", err)
	}
	return local, local, nil
}

func NewRouter(cfg config.Config, pool *pgxpool.Pool, svc Services, localFiles http.Handler) http.Handler {
	r := chi.NewRouter()

	if cfg.TrustProxy {
		r.Use(middleware.ClientIP(cfg.ClientIPHeader))
	}
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	r.Use(middleware.Cors(cfg.AllowedOrigins))
	r.Use(chimw.Timeout(cfg.WriteTimeout - time.Second))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { response.Error(w, r, response.ErrRouteNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { response.Error(w, r, response.ErrMethodNotAllowed) })

	controller.NewHealthController(r, pool)
	if localFiles != nil {
		r.With(middleware.RateLimitByIP(cfg.RateLimitPerMinute*10)).Get("/uploads/*", localFiles.ServeHTTP)
	}

	r.Route("/api/v1", func(v1 chi.Router) {
		// JSON endpoints: small bodies only.
		json := v1.With(middleware.BodyLimit(cfg.MaxBodyBytes))

		public := json.With(middleware.RateLimitByIP(cfg.RateLimitPerMinute))
		login := json.With(middleware.RateLimitByIP(cfg.LoginRateLimitPerMinute))

		admin := chi.NewRouter()
		admin.Use(middleware.RateLimitByIP(cfg.RateLimitPerMinute * 3))
		admin.Use(middleware.RequireAdmin(svc.Auth))
		adminJSON := admin.With(middleware.BodyLimit(cfg.MaxBodyBytes))
		// Uploads get a larger cap (+64KB for multipart framing).
		adminUpload := admin.With(middleware.BodyLimit(cfg.MaxUploadBytes + 64<<10))

		controller.NewAuthController(login, adminJSON, svc.Auth)
		controller.NewCvController(public, adminJSON, svc.Cv)
		controller.NewTripController(public, adminJSON, svc.Trip)
		controller.NewMilestoneController(public, adminJSON, svc.Milestone)
		controller.NewPostController(public, adminJSON, svc.Post)
		controller.NewUploadController(adminUpload, svc.Upload)

		v1.Mount("/admin", admin)
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
