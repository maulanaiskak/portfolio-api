package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type MilestoneController struct {
	milestoneService service.MilestoneService
}

// GET /api/v1/milestones?page=&limit=
func (c *MilestoneController) listPublished(w http.ResponseWriter, r *http.Request) {
	items, total, err := c.milestoneService.ListPublished(r.Context(), pageFromQuery(r))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewMilestoneSummaries(items), int64(total))
}

// GET /api/v1/milestones/{slug}
func (c *MilestoneController) getPublished(w http.ResponseWriter, r *http.Request) {
	m, err := c.milestoneService.GetPublished(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewMilestoneResponse(m))
}

// GET /api/v1/admin/milestones?page=&limit=&status=
func (c *MilestoneController) list(w http.ResponseWriter, r *http.Request) {
	status, err := statusFromQuery(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	items, total, err := c.milestoneService.List(r.Context(), pageFromQuery(r), status)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewMilestoneSummaries(items), int64(total))
}

// GET /api/v1/admin/milestones/{id}
func (c *MilestoneController) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	m, err := c.milestoneService.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewMilestoneResponse(m))
}

// POST /api/v1/admin/milestones
func (c *MilestoneController) create(w http.ResponseWriter, r *http.Request) {
	m, err := c.decode(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	m, err = c.milestoneService.Create(r.Context(), m)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, dto.NewMilestoneResponse(m))
}

// PUT /api/v1/admin/milestones/{id}
func (c *MilestoneController) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	m, err := c.decode(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	m.ID = id
	m, err = c.milestoneService.Update(r.Context(), m)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewMilestoneResponse(m))
}

// DELETE /api/v1/admin/milestones/{id}
func (c *MilestoneController) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	if err := c.milestoneService.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

func (c *MilestoneController) decode(r *http.Request) (entity.Milestone, error) {
	var req dto.MilestoneRequest
	if err := decodeJSON(r, &req); err != nil {
		return entity.Milestone{}, err
	}
	return req.ToEntity()
}

func NewMilestoneController(publicRouter, adminRouter chi.Router, milestoneService service.MilestoneService) *MilestoneController {
	c := &MilestoneController{milestoneService: milestoneService}
	publicRouter.Get("/milestones", c.listPublished)
	publicRouter.Get("/milestones/{slug}", c.getPublished)

	adminRouter.Route("/milestones", func(r chi.Router) {
		r.Get("/", c.list)
		r.Post("/", c.create)
		r.Get("/{id}", c.get)
		r.Put("/{id}", c.update)
		r.Delete("/{id}", c.delete)
	})
	return c
}
