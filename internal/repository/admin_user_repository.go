package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type AdminUserRepository interface {
	FindByEmail(ctx context.Context, email string) (entity.AdminUser, error)
	FindByID(ctx context.Context, id int64) (entity.AdminUser, error)
	// DeleteAllExcept removes every admin row except email (single-admin model).
	DeleteAllExcept(ctx context.Context, email string) (int64, error)
	// Upsert creates the admin row or rotates its password hash.
	Upsert(ctx context.Context, email, passwordHash string) (entity.AdminUser, error)
}

type adminUserRepository struct {
	pool *pgxpool.Pool
}

func (r *adminUserRepository) FindByEmail(ctx context.Context, email string) (entity.AdminUser, error) {
	var u entity.AdminUser
	err := conn(ctx, r.pool).QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM admin_users
		WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, entity.NewNotFoundError("admin user not found")
	}
	return u, err
}

func (r *adminUserRepository) FindByID(ctx context.Context, id int64) (entity.AdminUser, error) {
	var u entity.AdminUser
	err := conn(ctx, r.pool).QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM admin_users
		WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, entity.NewNotFoundError("admin user not found")
	}
	return u, err
}

func (r *adminUserRepository) DeleteAllExcept(ctx context.Context, email string) (int64, error) {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM admin_users WHERE email <> $1`, email)
	return tag.RowsAffected(), err
}

func (r *adminUserRepository) Upsert(ctx context.Context, email, passwordHash string) (entity.AdminUser, error) {
	var u entity.AdminUser
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO admin_users (email, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE
			SET password_hash = EXCLUDED.password_hash
			WHERE admin_users.password_hash IS DISTINCT FROM EXCLUDED.password_hash
		RETURNING id, email, password_hash, created_at, updated_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Conflict with an identical hash: nothing changed.
		return r.FindByEmail(ctx, email)
	}
	return u, err
}

func NewAdminUserRepository(pool *pgxpool.Pool) AdminUserRepository {
	return &adminUserRepository{pool: pool}
}
