package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type CvController struct {
	cvService service.CvService
}

// getCv returns experiences, projects, education and skills in one payload.
//
//	GET /api/v1/cv
func (c *CvController) getCv(w http.ResponseWriter, r *http.Request) {
	cv, err := c.cvService.GetCv(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	// Public, rarely-changing data: let the CDN / Next.js ISR cache briefly.
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	response.Success(w, dto.NewCvResponse(cv))
}

func NewCvController(publicRouter chi.Router, cvService service.CvService) *CvController {
	controller := &CvController{cvService: cvService}
	publicRouter.Get("/cv", controller.getCv)
	return controller
}
