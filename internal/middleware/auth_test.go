package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type fakeAuth struct{ validToken string }

func (f fakeAuth) Login(context.Context, string, string) (service.LoginResult, error) {
	return service.LoginResult{}, nil
}

func (f fakeAuth) Authenticate(_ context.Context, token string) (entity.AdminUser, error) {
	if token != f.validToken {
		return entity.AdminUser{}, entity.NewUnauthorizedError("Invalid or expired token")
	}
	return entity.AdminUser{ID: 7, Email: "admin@test"}, nil
}

func TestRequireAdmin(t *testing.T) {
	var gotAdmin entity.AdminUser
	handler := RequireAdmin(fakeAuth{validToken: "good"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAdmin, _ = AdminFromContext(r.Context())
		w.WriteHeader(http.StatusTeapot)
	}))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"basic scheme", "Basic Z29vZA==", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"bad token", "Bearer nope", http.StatusUnauthorized},
		{"valid token", "Bearer good", http.StatusTeapot},
		{"case-insensitive scheme", "bearer good", http.StatusTeapot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotAdmin = entity.AdminUser{}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, c.want, rec.Body)
			}
			if c.want == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("missing WWW-Authenticate header")
			}
			if c.want == http.StatusTeapot && gotAdmin.ID != 7 {
				t.Fatal("admin not placed in context")
			}
		})
	}
}
