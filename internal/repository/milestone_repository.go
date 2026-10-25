package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type MilestoneRepository interface {
	List(ctx context.Context, f entity.ListFilter) ([]entity.Milestone, int, error)
	FindByID(ctx context.Context, id int64) (entity.Milestone, error)
	FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Milestone, error)
	Create(ctx context.Context, m *entity.Milestone) error
	Update(ctx context.Context, m *entity.Milestone) error
	Delete(ctx context.Context, id int64) error
}

type milestoneRepository struct {
	pool *pgxpool.Pool
}

const milestoneColumns = `id, slug, title, company, role, milestone_date, summary, body_markdown,
	COALESCE(cover_image_url, ''), status, published_at, created_at, updated_at`

// Career reads as a timeline: most recent milestone first.
const milestoneOrder = `ORDER BY milestone_date DESC NULLS LAST, published_at DESC NULLS LAST, id DESC`

func scanMilestone(row pgx.Row) (entity.Milestone, error) {
	var m entity.Milestone
	err := row.Scan(&m.ID, &m.Slug, &m.Title, &m.Company, &m.Role, &m.MilestoneDate, &m.Summary,
		&m.BodyMarkdown, &m.CoverImageURL, &m.Status, &m.PublishedAt, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *milestoneRepository) List(ctx context.Context, f entity.ListFilter) ([]entity.Milestone, int, error) {
	db := conn(ctx, r.pool)
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM career_milestones WHERE ($1::text IS NULL OR status = $1)`, f.Status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(ctx, `SELECT `+milestoneColumns+` FROM career_milestones
		WHERE ($1::text IS NULL OR status = $1) `+milestoneOrder+` LIMIT $2 OFFSET $3`,
		f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Milestone, error) { return scanMilestone(row) })
	return items, total, err
}

func (r *milestoneRepository) FindByID(ctx context.Context, id int64) (entity.Milestone, error) {
	m, err := scanMilestone(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+milestoneColumns+` FROM career_milestones WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, entity.NewNotFoundError("milestone not found")
	}
	return m, err
}

func (r *milestoneRepository) FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Milestone, error) {
	m, err := scanMilestone(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+milestoneColumns+` FROM career_milestones
		WHERE slug = $1 AND ($2::text IS NULL OR status = $2)`, slug, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, entity.NewNotFoundError("milestone not found")
	}
	return m, err
}

func (r *milestoneRepository) Create(ctx context.Context, m *entity.Milestone) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO career_milestones (slug, title, company, role, milestone_date, summary,
			body_markdown, cover_image_url, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		m.Slug, m.Title, m.Company, m.Role, m.MilestoneDate, m.Summary,
		m.BodyMarkdown, nullIfEmpty(m.CoverImageURL), m.Status, m.PublishedAt,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	return mapWriteError(err, "milestone")
}

func (r *milestoneRepository) Update(ctx context.Context, m *entity.Milestone) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE career_milestones SET slug = $2, title = $3, company = $4, role = $5,
			milestone_date = $6, summary = $7, body_markdown = $8, cover_image_url = $9,
			status = $10, published_at = $11
		WHERE id = $1
		RETURNING created_at, updated_at`,
		m.ID, m.Slug, m.Title, m.Company, m.Role, m.MilestoneDate, m.Summary,
		m.BodyMarkdown, nullIfEmpty(m.CoverImageURL), m.Status, m.PublishedAt,
	).Scan(&m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.NewNotFoundError("milestone not found")
	}
	return mapWriteError(err, "milestone")
}

func (r *milestoneRepository) Delete(ctx context.Context, id int64) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM career_milestones WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.NewNotFoundError("milestone not found")
	}
	return nil
}

func NewMilestoneRepository(pool *pgxpool.Pool) MilestoneRepository {
	return &milestoneRepository{pool: pool}
}
