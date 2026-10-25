package service

import (
	"context"
	"strings"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

type AdminUserService interface {
	// EnsureAdmin creates the single admin row, or rotates its password hash
	// when ADMIN_PASSWORD_BCRYPT_HASH changes. Only bcrypt hashes are accepted.
	EnsureAdmin(ctx context.Context, email, passwordBcryptHash string) (entity.AdminUser, error)
}

type adminUserService struct {
	adminRepo repository.AdminUserRepository
}

func (s *adminUserService) EnsureAdmin(ctx context.Context, email, passwordBcryptHash string) (entity.AdminUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.HasPrefix(passwordBcryptHash, "$2") {
		return entity.AdminUser{}, entity.NewValidationError("admin email and bcrypt hash are required")
	}
	return s.adminRepo.Upsert(ctx, email, passwordBcryptHash)
}

func NewAdminUserService(adminRepo repository.AdminUserRepository) AdminUserService {
	return &adminUserService{adminRepo: adminRepo}
}
