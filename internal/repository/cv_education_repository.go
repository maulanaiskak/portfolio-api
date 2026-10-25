package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type CvEducationRepository interface {
	FindAll(ctx context.Context) ([]entity.Education, error)
	Create(ctx context.Context, e *entity.Education) error
	DeleteAll(ctx context.Context) error
	Count(ctx context.Context) (int, error)
}

type cvEducationRepository struct {
	pool *pgxpool.Pool
}

func (r *cvEducationRepository) FindAll(ctx context.Context) ([]entity.Education, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT id, institution, location, degree, period, gpa, thesis, thesis_link, sort_order
		FROM cv_education
		ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Education, error) {
		var e entity.Education
		err := row.Scan(&e.ID, &e.Institution, &e.Location, &e.Degree, &e.Period,
			&e.GPA, &e.Thesis, &e.ThesisLink, &e.SortOrder)
		return e, err
	})
}

func (r *cvEducationRepository) Create(ctx context.Context, e *entity.Education) error {
	return conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cv_education (institution, location, degree, period, gpa, thesis, thesis_link, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		e.Institution, e.Location, e.Degree, e.Period, e.GPA, e.Thesis, e.ThesisLink, e.SortOrder,
	).Scan(&e.ID)
}

func (r *cvEducationRepository) DeleteAll(ctx context.Context) error {
	_, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM cv_education`)
	return err
}

func (r *cvEducationRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM cv_education`).Scan(&n)
	return n, err
}

func NewCvEducationRepository(pool *pgxpool.Pool) CvEducationRepository {
	return &cvEducationRepository{pool: pool}
}
