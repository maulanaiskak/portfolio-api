package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type TripController struct {
	tripService service.TripService
}

// GET /api/v1/trips?page=&limit=
func (c *TripController) listPublished(w http.ResponseWriter, r *http.Request) {
	trips, total, err := c.tripService.ListPublished(r.Context(), pageFromQuery(r))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewTripSummaries(trips), int64(total))
}

// GET /api/v1/trips/{slug}
func (c *TripController) getPublished(w http.ResponseWriter, r *http.Request) {
	trip, err := c.tripService.GetPublished(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewTripResponse(trip))
}

// GET /api/v1/admin/trips?page=&limit=&status=
func (c *TripController) list(w http.ResponseWriter, r *http.Request) {
	status, err := statusFromQuery(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	trips, total, err := c.tripService.List(r.Context(), pageFromQuery(r), status)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewTripSummaries(trips), int64(total))
}

// GET /api/v1/admin/trips/{id}
func (c *TripController) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	trip, err := c.tripService.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewTripResponse(trip))
}

// POST /api/v1/admin/trips
func (c *TripController) create(w http.ResponseWriter, r *http.Request) {
	trip, err := c.decode(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	trip, err = c.tripService.Create(r.Context(), trip)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, dto.NewTripResponse(trip))
}

// PUT /api/v1/admin/trips/{id}
func (c *TripController) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	trip, err := c.decode(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	trip.ID = id
	trip, err = c.tripService.Update(r.Context(), trip)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewTripResponse(trip))
}

// DELETE /api/v1/admin/trips/{id}
func (c *TripController) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	if err := c.tripService.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

// POST /api/v1/admin/trips/{id}/photos
func (c *TripController) addPhoto(w http.ResponseWriter, r *http.Request) {
	tripID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	var req dto.TripPhotoRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	photo, err := c.tripService.AddPhoto(r.Context(), entity.TripPhoto{
		TripID: tripID, URL: req.URL, Caption: req.Caption, SortOrder: req.SortOrder,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, dto.NewTripPhotoResponse(photo))
}

// PUT /api/v1/admin/trips/{id}/photos/{photoId} (caption and order only)
func (c *TripController) updatePhoto(w http.ResponseWriter, r *http.Request) {
	tripID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	photoID, err := pathID(r, "photoId")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	var req dto.TripPhotoRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	photo, err := c.tripService.UpdatePhoto(r.Context(), entity.TripPhoto{
		ID: photoID, TripID: tripID, Caption: req.Caption, SortOrder: req.SortOrder,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewTripPhotoResponse(photo))
}

// DELETE /api/v1/admin/trips/{id}/photos/{photoId}
func (c *TripController) deletePhoto(w http.ResponseWriter, r *http.Request) {
	tripID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	photoID, err := pathID(r, "photoId")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	if err := c.tripService.DeletePhoto(r.Context(), tripID, photoID); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

func (c *TripController) decode(r *http.Request) (entity.Trip, error) {
	var req dto.TripRequest
	if err := decodeJSON(r, &req); err != nil {
		return entity.Trip{}, err
	}
	return req.ToEntity()
}

func NewTripController(publicRouter, adminRouter chi.Router, tripService service.TripService) *TripController {
	c := &TripController{tripService: tripService}
	publicRouter.Get("/trips", c.listPublished)
	publicRouter.Get("/trips/{slug}", c.getPublished)

	adminRouter.Route("/trips", func(r chi.Router) {
		r.Get("/", c.list)
		r.Post("/", c.create)
		r.Get("/{id}", c.get)
		r.Put("/{id}", c.update)
		r.Delete("/{id}", c.delete)
		r.Post("/{id}/photos", c.addPhoto)
		r.Put("/{id}/photos/{photoId}", c.updatePhoto)
		r.Delete("/{id}/photos/{photoId}", c.deletePhoto)
	})
	return c
}
