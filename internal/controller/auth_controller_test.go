package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

// Template for controller tests: real router + controller, fake service.

type fakeAuthService struct{}

func (fakeAuthService) Login(_ context.Context, email, password string) (service.LoginResult, error) {
	if email == "admin@test" && password == "correct horse" {
		return service.LoginResult{Token: "jwt", ExpiresAt: time.Unix(0, 0), Admin: entity.AdminUser{ID: 1, Email: email}}, nil
	}
	return service.LoginResult{}, entity.NewUnauthorizedError("Invalid email or password")
}

func (fakeAuthService) Authenticate(context.Context, string) (entity.AdminUser, error) {
	return entity.AdminUser{}, entity.NewUnauthorizedError("unused")
}

func TestLogin(t *testing.T) {
	r := chi.NewRouter()
	NewAuthController(r, chi.NewRouter(), fakeAuthService{})

	cases := []struct {
		name     string
		body     string
		wantCode int
		wantResp string
	}{
		{"success", `{"email":"admin@test","password":"correct horse"}`, http.StatusOK, "00"},
		{"wrong password", `{"email":"admin@test","password":"nope"}`, http.StatusUnauthorized, "E04"},
		{"unknown field", `{"email":"a","password":"b","admin":true}`, http.StatusBadRequest, "H05"},
		{"malformed", `{"email":`, http.StatusBadRequest, "H05"},
		{"empty", ``, http.StatusBadRequest, "H06"},
		{"two objects", `{"email":"a","password":"b"}{}`, http.StatusBadRequest, "H05"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			var resp struct {
				Code string `json:"response_code"`
				Data *struct {
					Token     string `json:"token"`
					TokenType string `json:"tokenType"`
				} `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("invalid JSON response: %s", rec.Body)
			}
			if rec.Code != c.wantCode || resp.Code != c.wantResp {
				t.Fatalf("got %d/%s, want %d/%s", rec.Code, resp.Code, c.wantCode, c.wantResp)
			}
			if c.wantCode == http.StatusOK && (resp.Data == nil || resp.Data.Token != "jwt" || resp.Data.TokenType != "Bearer") {
				t.Fatalf("unexpected body: %s", rec.Body)
			}
		})
	}
}
