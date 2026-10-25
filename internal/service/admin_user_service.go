package service

import (
	"context"
	"strings"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

type AdminUserService interface {
	// EnsureAdmin makes email the single admin: it creates the row (or rotates
	// its password hash when ADMIN_PASSWORD_BCRYPT_HASH changes) and removes
	// any other admin rows. Only bcrypt hashes are accepted.
	EnsureAdmin(ctx context.Context, email, passwordBcryptHash string) (entity.AdminUser, error)
}

type adminUserService struct {
	adminRepo repository.AdminUserRepository
	tx        repository.Transactor
}

func (s *adminUserService) EnsureAdmin(ctx context.Context, email, passwordBcryptHash string) (entity.AdminUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.HasPrefix(passwordBcryptHash, "$2") {
		return entity.AdminUser{}, entity.NewValidationError("admin email and bcrypt hash are required")
	}
	var admin entity.AdminUser
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if admin, err = s.adminRepo.Upsert(ctx, email, passwordBcryptHash); err != nil {
			return err
		}
		_, err = s.adminRepo.DeleteAllExcept(ctx, email)
		return err
	})
	return admin, err
}

func NewAdminUserService(adminRepo repository.AdminUserRepository, tx repository.Transactor) AdminUserService {
	return &adminUserService{adminRepo: adminRepo, tx: tx}
}
