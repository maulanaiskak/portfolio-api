package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type CvProjectRepository interface {
	FindAll(ctx context.Context) ([]entity.Project, error)
	Create(ctx context.Context, p *entity.Project) error
	DeleteAll(ctx context.Context) error
	Count(ctx context.Context) (int, error)
}

type cvProjectRepository struct {
	pool *pgxpool.Pool
}

func (r *cvProjectRepository) FindAll(ctx context.Context) ([]entity.Project, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT id, title, description, technologies, image, highlights, sort_order
		FROM cv_projects
		ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Project, error) {
		var p entity.Project
		err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Technologies, &p.Image, &p.Highlights, &p.SortOrder)
		return p, err
	})
}

func (r *cvProjectRepository) Create(ctx context.Context, p *entity.Project) error {
	return conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cv_projects (title, description, technologies, image, highlights, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		p.Title, p.Description, nonNil(p.Technologies), p.Image, nonNil(p.Highlights), p.SortOrder,
	).Scan(&p.ID)
}

func (r *cvProjectRepository) DeleteAll(ctx context.Context) error {
	_, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM cv_projects`)
	return err
}

func (r *cvProjectRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM cv_projects`).Scan(&n)
	return n, err
}

func NewCvProjectRepository(pool *pgxpool.Pool) CvProjectRepository {
	return &cvProjectRepository{pool: pool}
}
