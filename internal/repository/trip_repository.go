package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type TripRepository interface {
	List(ctx context.Context, f entity.ListFilter) ([]entity.Trip, int, error)
	FindByID(ctx context.Context, id int64) (entity.Trip, error)
	FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Trip, error)
	Create(ctx context.Context, t *entity.Trip) error
	Update(ctx context.Context, t *entity.Trip) error
	Delete(ctx context.Context, id int64) error

	ListPhotos(ctx context.Context, tripID int64) ([]entity.TripPhoto, error)
	CreatePhoto(ctx context.Context, p *entity.TripPhoto) error
	UpdatePhoto(ctx context.Context, p *entity.TripPhoto) error
	DeletePhoto(ctx context.Context, tripID, photoID int64) error
}

type tripRepository struct {
	pool *pgxpool.Pool
}

const tripColumns = `id, slug, title, location, COALESCE(country_code, ''), start_date, end_date,
	summary, body_markdown, COALESCE(cover_image_url, ''), status, published_at, created_at, updated_at`

// Public listing order: newest trip first, then most recently published.
const tripOrder = `ORDER BY start_date DESC NULLS LAST, published_at DESC NULLS LAST, id DESC`

func scanTrip(row pgx.Row) (entity.Trip, error) {
	var t entity.Trip
	err := row.Scan(&t.ID, &t.Slug, &t.Title, &t.Location, &t.CountryCode, &t.StartDate, &t.EndDate,
		&t.Summary, &t.BodyMarkdown, &t.CoverImageURL, &t.Status, &t.PublishedAt, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *tripRepository) List(ctx context.Context, f entity.ListFilter) ([]entity.Trip, int, error) {
	db := conn(ctx, r.pool)
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM trips WHERE ($1::text IS NULL OR status = $1)`, f.Status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(ctx, `SELECT `+tripColumns+` FROM trips
		WHERE ($1::text IS NULL OR status = $1) `+tripOrder+` LIMIT $2 OFFSET $3`,
		f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	trips, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Trip, error) { return scanTrip(row) })
	return trips, total, err
}

func (r *tripRepository) FindByID(ctx context.Context, id int64) (entity.Trip, error) {
	t, err := scanTrip(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+tripColumns+` FROM trips WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, entity.NewNotFoundError("trip not found")
	}
	return t, err
}

func (r *tripRepository) FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Trip, error) {
	t, err := scanTrip(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+tripColumns+` FROM trips
		WHERE slug = $1 AND ($2::text IS NULL OR status = $2)`, slug, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, entity.NewNotFoundError("trip not found")
	}
	return t, err
}

func (r *tripRepository) Create(ctx context.Context, t *entity.Trip) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO trips (slug, title, location, country_code, start_date, end_date,
			summary, body_markdown, cover_image_url, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`,
		t.Slug, t.Title, t.Location, nullIfEmpty(t.CountryCode), t.StartDate, t.EndDate,
		t.Summary, t.BodyMarkdown, nullIfEmpty(t.CoverImageURL), t.Status, t.PublishedAt,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	return mapWriteError(err, "trip")
}

func (r *tripRepository) Update(ctx context.Context, t *entity.Trip) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE trips SET slug = $2, title = $3, location = $4, country_code = $5, start_date = $6,
			end_date = $7, summary = $8, body_markdown = $9, cover_image_url = $10, status = $11,
			published_at = $12
		WHERE id = $1
		RETURNING created_at, updated_at`,
		t.ID, t.Slug, t.Title, t.Location, nullIfEmpty(t.CountryCode), t.StartDate, t.EndDate,
		t.Summary, t.BodyMarkdown, nullIfEmpty(t.CoverImageURL), t.Status, t.PublishedAt,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.NewNotFoundError("trip not found")
	}
	return mapWriteError(err, "trip")
}

func (r *tripRepository) Delete(ctx context.Context, id int64) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM trips WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.NewNotFoundError("trip not found")
	}
	return nil
}

func (r *tripRepository) ListPhotos(ctx context.Context, tripID int64) ([]entity.TripPhoto, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT id, trip_id, url, caption, sort_order, created_at
		FROM trip_photos WHERE trip_id = $1
		ORDER BY sort_order, id`, tripID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.TripPhoto, error) {
		var p entity.TripPhoto
		err := row.Scan(&p.ID, &p.TripID, &p.URL, &p.Caption, &p.SortOrder, &p.CreatedAt)
		return p, err
	})
}

func (r *tripRepository) CreatePhoto(ctx context.Context, p *entity.TripPhoto) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO trip_photos (trip_id, url, caption, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		p.TripID, p.URL, p.Caption, p.SortOrder,
	).Scan(&p.ID, &p.CreatedAt)
	return mapWriteError(err, "trip")
}

func (r *tripRepository) UpdatePhoto(ctx context.Context, p *entity.TripPhoto) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE trip_photos SET caption = $3, sort_order = $4
		WHERE id = $1 AND trip_id = $2
		RETURNING url, created_at`,
		p.ID, p.TripID, p.Caption, p.SortOrder,
	).Scan(&p.URL, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.NewNotFoundError("photo not found")
	}
	return mapWriteError(err, "photo")
}

func (r *tripRepository) DeletePhoto(ctx context.Context, tripID, photoID int64) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM trip_photos WHERE id = $1 AND trip_id = $2`, photoID, tripID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.NewNotFoundError("photo not found")
	}
	return nil
}

func NewTripRepository(pool *pgxpool.Pool) TripRepository {
	return &tripRepository{pool: pool}
}
