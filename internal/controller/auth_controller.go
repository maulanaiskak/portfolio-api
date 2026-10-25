package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/middleware"
	"github.com/maulanaiskak/portfolio-api/internal/response"
	"github.com/maulanaiskak/portfolio-api/internal/service"
)

type AuthController struct {
	authService service.AuthService
}

// login exchanges admin credentials for a JWT.
//
//	POST /api/v1/auth/login
func (c *AuthController) login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	result, err := c.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, dto.LoginResponse{
		Token: result.Token, TokenType: "Bearer", ExpiresAt: result.ExpiresAt,
		Admin: dto.NewAdminResponse(result.Admin),
	})
}

// me returns the authenticated admin; the web app uses it to validate a
// stored token.
//
//	GET /api/v1/admin/me
func (c *AuthController) me(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.AdminFromContext(r.Context())
	response.Success(w, dto.NewAdminResponse(admin))
}

func NewAuthController(loginRouter, adminRouter chi.Router, authService service.AuthService) *AuthController {
	c := &AuthController{authService: authService}
	loginRouter.Post("/auth/login", c.login)
	adminRouter.Get("/me", c.me)
	return c
}
