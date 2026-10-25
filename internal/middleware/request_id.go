package middleware

import (
	"context"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// RequestID assigns every request a random ID (exposed as X-Request-Id for
// support/debugging). Unlike chi's default it doesn't embed the hostname,
// and client-supplied IDs are ignored so logs can't be spoofed.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), chimw.RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
