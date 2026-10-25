package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

// decodeJSON decodes a single JSON object strictly: unknown fields and
// trailing data are rejected. Body size is capped by middleware.BodyLimit.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return response.ErrPayloadTooLarge
		case errors.Is(err, io.EOF):
			return response.ErrEmptyBody
		default:
			return response.ErrMalformedJSON
		}
	}
	if dec.More() {
		return response.ErrMalformedJSON
	}
	return nil
}

// pathID parses a positive int64 path parameter.
func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		return 0, entity.NewNotFoundError("resource not found")
	}
	return id, nil
}

// pageFromQuery reads ?page=&limit=, clamped by service.NewPage.
func pageFromQuery(r *http.Request) service.Page {
	q := r.URL.Query()
	number, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("limit"))
	return service.NewPage(number, size)
}

// statusFromQuery reads an optional ?status=draft|published filter.
func statusFromQuery(r *http.Request) (*entity.Status, error) {
	raw := r.URL.Query().Get("status")
	if raw == "" {
		return nil, nil
	}
	s := entity.Status(raw)
	if !s.Valid() {
		return nil, entity.NewValidationError("status must be draft or published")
	}
	return &s, nil
}

// publicCache marks public read responses as briefly cacheable by CDNs and
// Next.js ISR; admin responses keep the default no-store.
func publicCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
}
