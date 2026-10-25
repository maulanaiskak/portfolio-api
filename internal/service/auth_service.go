package service

import (
	"context"
	"strings"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
	"github.com/maulanaiskak/portfolio-api/internal/security"
)

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Admin     entity.AdminUser
}

type AuthService interface {
	Login(ctx context.Context, email, password string) (LoginResult, error)
	// Authenticate verifies a bearer token and returns the admin it belongs
	// to. Tokens are rejected once the admin's password has been rotated.
	Authenticate(ctx context.Context, token string) (entity.AdminUser, error)
}

type authService struct {
	adminRepo repository.AdminUserRepository
	tokens    security.TokenManager
}

// errInvalidCredentials is deliberately the same for unknown email and wrong
// password, so the response doesn't reveal which accounts exist.
func errInvalidCredentials() error {
	return entity.NewUnauthorizedError("Invalid email or password")
}

func (s *authService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" || len(password) > 72 {
		// bcrypt only uses the first 72 bytes; longer input is rejected
		// rather than silently truncated.
		security.ComparePassword("", password)
		return LoginResult{}, errInvalidCredentials()
	}

	admin, err := s.adminRepo.FindByEmail(ctx, email)
	if err != nil && !entity.IsKind(err, entity.KindNotFound) {
		return LoginResult{}, err
	}
	if !security.ComparePassword(admin.PasswordHash, password) {
		return LoginResult{}, errInvalidCredentials()
	}

	token, exp, err := s.tokens.Issue(admin.ID, security.PasswordVersion(admin.PasswordHash))
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: exp, Admin: admin}, nil
}

func (s *authService) Authenticate(ctx context.Context, token string) (entity.AdminUser, error) {
	unauthorized := entity.NewUnauthorizedError("Invalid or expired token")

	claims, err := s.tokens.Verify(token)
	if err != nil {
		return entity.AdminUser{}, unauthorized
	}
	id, err := claims.AdminID()
	if err != nil {
		return entity.AdminUser{}, unauthorized
	}
	admin, err := s.adminRepo.FindByID(ctx, id)
	if entity.IsKind(err, entity.KindNotFound) {
		return entity.AdminUser{}, unauthorized
	}
	if err != nil {
		return entity.AdminUser{}, err
	}
	if claims.PasswordVersion != security.PasswordVersion(admin.PasswordHash) {
		return entity.AdminUser{}, unauthorized
	}
	return admin, nil
}

func NewAuthService(adminRepo repository.AdminUserRepository, tokens security.TokenManager) AuthService {
	return &authService{adminRepo: adminRepo, tokens: tokens}
}
