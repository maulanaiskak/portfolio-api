package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// Cors allows only the exact configured origins. Credentials stay off since
// the admin JWT travels in the Authorization header, not a cookie.
func Cors(allowedOrigins []string) func(http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           600,
	}).Handler
}
