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
//	GET /api/v1/cv          (public, cacheable)
//	GET /api/v1/admin/cv    (admin, no-store)
func (c *CvController) getCv(public bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cv, err := c.cvService.GetCv(r.Context())
		if err != nil {
			response.Error(w, r, err)
			return
		}
		if public {
			publicCache(w)
		}
		response.Success(w, dto.NewCvResponse(cv))
	}
}

// replaceCv replaces the whole CV in one transaction (the admin editor
// saves the full document).
//
//	PUT /api/v1/admin/cv
func (c *CvController) replaceCv(w http.ResponseWriter, r *http.Request) {
	var req dto.CvRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	if _, err := c.cvService.ImportCv(r.Context(), req.ToEntity(), true); err != nil {
		response.Error(w, r, err)
		return
	}
	c.getCv(false)(w, r)
}

func NewCvController(publicRouter, adminRouter chi.Router, cvService service.CvService) *CvController {
	c := &CvController{cvService: cvService}
	publicRouter.Get("/cv", c.getCv(true))
	adminRouter.Get("/cv", c.getCv(false))
	adminRouter.Put("/cv", c.replaceCv)
	return c
}
