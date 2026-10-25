package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/maulanaiskak/portfolio-api/internal/response"
)

// decodeJSON decodes a single JSON object strictly: unknown fields and
// trailing data are rejected. Body size is capped by middleware.BodyLimit.
// Used by the admin write endpoints (Phase 2).
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
