package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type PostController struct {
	postService service.PostService
}

// GET /api/v1/posts?page=&limit=&tag=
func (c *PostController) listPublished(w http.ResponseWriter, r *http.Request) {
	items, total, err := c.postService.ListPublished(r.Context(), pageFromQuery(r), r.URL.Query().Get("tag"))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewPostSummaries(items), int64(total))
}

// GET /api/v1/posts/tags
func (c *PostController) tags(w http.ResponseWriter, r *http.Request) {
	tags, err := c.postService.PublishedTags(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewTagCounts(tags))
}

// GET /api/v1/posts/{slug}
func (c *PostController) getPublished(w http.ResponseWriter, r *http.Request) {
	p, err := c.postService.GetPublished(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	publicCache(w)
	response.Success(w, dto.NewPostResponse(p))
}

// GET /api/v1/admin/posts?page=&limit=&status=
func (c *PostController) list(w http.ResponseWriter, r *http.Request) {
	status, err := statusFromQuery(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	items, total, err := c.postService.List(r.Context(), pageFromQuery(r), status)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewPostSummaries(items), int64(total))
}

// GET /api/v1/admin/posts/{id}
func (c *PostController) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	p, err := c.postService.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(p))
}

// POST /api/v1/admin/posts
func (c *PostController) create(w http.ResponseWriter, r *http.Request) {
	var req dto.PostRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	p, err := c.postService.Create(r.Context(), req.ToEntity())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, dto.NewPostResponse(p))
}

// PUT /api/v1/admin/posts/{id}
func (c *PostController) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	var req dto.PostRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	p := req.ToEntity()
	p.ID = id
	p, err = c.postService.Update(r.Context(), p)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(p))
}

// DELETE /api/v1/admin/posts/{id}
func (c *PostController) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}
	if err := c.postService.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

func NewPostController(publicRouter, adminRouter chi.Router, postService service.PostService) *PostController {
	c := &PostController{postService: postService}
	publicRouter.Get("/posts", c.listPublished)
	publicRouter.Get("/posts/tags", c.tags)
	publicRouter.Get("/posts/{slug}", c.getPublished)

	adminRouter.Route("/posts", func(r chi.Router) {
		r.Get("/", c.list)
		r.Post("/", c.create)
		r.Get("/{id}", c.get)
		r.Put("/{id}", c.update)
		r.Delete("/{id}", c.delete)
	})
	return c
}
