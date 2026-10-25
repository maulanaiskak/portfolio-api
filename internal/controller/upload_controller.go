package controller

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type UploadController struct {
	uploadService service.UploadService
}

// uploadImage accepts multipart/form-data with a single "file" field and
// returns the stored image's public URL.
//
//	POST /api/v1/admin/uploads
func (c *UploadController) uploadImage(w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()
	if err != nil {
		response.Error(w, r, entity.NewValidationError("request must be multipart/form-data"))
		return
	}
	// Stream parts instead of ParseMultipartForm so nothing is spooled to
	// temp files; total size is capped by the route's BodyLimit.
	for {
		part, err := reader.NextPart()
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				response.Error(w, r, response.ErrPayloadTooLarge)
				return
			}
			response.Error(w, r, entity.NewValidationError(`multipart field "file" is required`))
			return
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		c.store(w, r, part)
		return
	}
}

func (c *UploadController) store(w http.ResponseWriter, r *http.Request, part *multipart.Part) {
	defer part.Close()
	img, err := c.uploadService.UploadImage(r.Context(), part)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			err = response.ErrPayloadTooLarge
		}
		response.Error(w, r, err)
		return
	}
	response.Created(w, img)
}

func NewUploadController(adminRouter chi.Router, uploadService service.UploadService) *UploadController {
	c := &UploadController{uploadService: uploadService}
	adminRouter.Post("/uploads", c.uploadImage)
	return c
}
