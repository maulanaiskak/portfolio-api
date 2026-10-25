package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type adminKey struct{}

// RequireAdmin rejects requests without a valid "Authorization: Bearer
// <jwt>" header and puts the authenticated admin in the request context.
func RequireAdmin(auth service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="admin"`)
				response.Error(w, r, entity.NewUnauthorizedError("Missing bearer token"))
				return
			}
			admin, err := auth.Authenticate(r.Context(), strings.TrimSpace(token))
			if err != nil {
				if entity.IsKind(err, entity.KindUnauthorized) {
					w.Header().Set("WWW-Authenticate", `Bearer realm="admin", error="invalid_token"`)
				}
				response.Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), adminKey{}, admin)))
		})
	}
}

// AdminFromContext returns the admin set by RequireAdmin.
func AdminFromContext(ctx context.Context) (entity.AdminUser, bool) {
	a, ok := ctx.Value(adminKey{}).(entity.AdminUser)
	return a, ok
}
