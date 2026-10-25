package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type CvExperienceRepository interface {
	FindAll(ctx context.Context) ([]entity.Experience, error)
	Create(ctx context.Context, exp *entity.Experience) error
	DeleteAll(ctx context.Context) error
	Count(ctx context.Context) (int, error)
}

type cvExperienceRepository struct {
	pool *pgxpool.Pool
}

func (r *cvExperienceRepository) FindAll(ctx context.Context) ([]entity.Experience, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT id, company, location, position, period,
		       responsibilities, projects_and_achievements, tech, sort_order
		FROM cv_experiences
		ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Experience, error) {
		var e entity.Experience
		err := row.Scan(&e.ID, &e.Company, &e.Location, &e.Position, &e.Period,
			&e.Responsibilities, &e.ProjectsAndAchievements, &e.Tech, &e.SortOrder)
		return e, err
	})
}

func (r *cvExperienceRepository) Create(ctx context.Context, e *entity.Experience) error {
	return conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cv_experiences
			(company, location, position, period, responsibilities, projects_and_achievements, tech, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		e.Company, e.Location, e.Position, e.Period,
		nonNil(e.Responsibilities), nonNil(e.ProjectsAndAchievements), nonNil(e.Tech), e.SortOrder,
	).Scan(&e.ID)
}

func (r *cvExperienceRepository) DeleteAll(ctx context.Context) error {
	_, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM cv_experiences`)
	return err
}

func (r *cvExperienceRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM cv_experiences`).Scan(&n)
	return n, err
}

func NewCvExperienceRepository(pool *pgxpool.Pool) CvExperienceRepository {
	return &cvExperienceRepository{pool: pool}
}
