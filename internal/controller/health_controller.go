package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/response"
)

// Pinger is the only thing the health check needs from the database.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthController struct {
	db Pinger
}

// healthz is the liveness/readiness probe for the host (Render health checks).
// It reports only up/down, with no version or dependency details.
//
//	GET /healthz
func (h *HealthController) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		response.Error(w, r, response.ErrUnavailable)
		return
	}
	response.Success(w, map[string]string{"status": "ok"})
}

func NewHealthController(router chi.Router, db Pinger) *HealthController {
	controller := &HealthController{db: db}
	router.Get("/healthz", controller.healthz)
	return controller
}
