package dto

import (
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type LoginResponse struct {
	Token     string        `json:"token"`
	TokenType string        `json:"tokenType"`
	ExpiresAt time.Time     `json:"expiresAt"`
	Admin     AdminResponse `json:"admin"`
}

func NewAdminResponse(a entity.AdminUser) AdminResponse {
	return AdminResponse{ID: a.ID, Email: a.Email}
}
