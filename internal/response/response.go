// Package response writes the API's JSON envelope and maps errors to HTTP
// status codes. Shared by controllers and middleware.
package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

const (
	SuccessCode         = "00"
	SuccessMessage      = "Success"
	DefaultErrorCode    = "XX"
	DefaultErrorMessage = "Something went wrong"
)

type Status struct {
	ResponseCode    string `json:"response_code"`
	ResponseMessage string `json:"response_message"`
}

type Response struct {
	Status
	Data      any    `json:"data,omitempty"`
	TotalData *int64 `json:"total_data,omitempty"` // set for list endpoints, even when 0
}

// HTTPError is a transport-level error (body too large, rate limited, ...)
// that has no business meaning, so it lives here rather than in entity.
type HTTPError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *HTTPError) Error() string { return e.Message }

var (
	ErrPayloadTooLarge  = &HTTPError{http.StatusRequestEntityTooLarge, "H01", "Request body is too large"}
	ErrTooManyRequests  = &HTTPError{http.StatusTooManyRequests, "H02", "Too many requests"}
	ErrRouteNotFound    = &HTTPError{http.StatusNotFound, "H03", "Route not found"}
	ErrMethodNotAllowed = &HTTPError{http.StatusMethodNotAllowed, "H04", "Method not allowed"}
	ErrMalformedJSON    = &HTTPError{http.StatusBadRequest, "H05", "Malformed JSON body"}
	ErrEmptyBody        = &HTTPError{http.StatusBadRequest, "H06", "Request body cannot be empty"}
	ErrUnavailable      = &HTTPError{http.StatusServiceUnavailable, "H07", "Service unavailable"}
)

var entityErrorStatus = map[entity.ErrorKind]struct {
	status int
	code   string
}{
	entity.KindNotFound:     {http.StatusNotFound, "E01"},
	entity.KindValidation:   {http.StatusBadRequest, "E02"},
	entity.KindConflict:     {http.StatusConflict, "E03"},
	entity.KindUnauthorized: {http.StatusUnauthorized, "E04"},
}

func Success(w http.ResponseWriter, data any, totalData ...int64) {
	resp := Response{
		Status: Status{ResponseCode: SuccessCode, ResponseMessage: SuccessMessage},
		Data:   data,
	}
	if len(totalData) != 0 {
		resp.TotalData = &totalData[0]
	}
	writeJSON(w, http.StatusOK, resp)
}

// Created writes a 201 with the success envelope.
func Created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, Response{
		Status: Status{ResponseCode: SuccessCode, ResponseMessage: SuccessMessage},
		Data:   data,
	})
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes err as the envelope. Entity and HTTP errors carry
// client-safe messages; anything else is logged and returned as a generic
// 500 so DB errors and internals never leak.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var (
		httpErr   *HTTPError
		entityErr *entity.Error
	)
	switch {
	case errors.As(err, &httpErr):
		writeJSON(w, httpErr.StatusCode, Response{Status: Status{httpErr.Code, httpErr.Message}})
	case errors.As(err, &entityErr):
		m, ok := entityErrorStatus[entityErr.Kind]
		if !ok {
			m.status, m.code = http.StatusBadRequest, "E00"
		}
		writeJSON(w, m.status, Response{Status: Status{m.code, entityErr.Message}})
	default:
		slog.ErrorContext(r.Context(), "internal error", "err", err, "method", r.Method, "path", r.URL.Path)
		writeJSON(w, http.StatusInternalServerError, Response{Status: Status{DefaultErrorCode, DefaultErrorMessage}})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write response", "err", err)
	}
}
