package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type PostRepository interface {
	List(ctx context.Context, f entity.ListFilter) ([]entity.Post, int, error)
	FindByID(ctx context.Context, id int64) (entity.Post, error)
	FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Post, error)
	Create(ctx context.Context, p *entity.Post) error
	Update(ctx context.Context, p *entity.Post) error
	Delete(ctx context.Context, id int64) error
	// Tags counts tags across posts with the given status (nil = all).
	Tags(ctx context.Context, status *entity.Status) ([]entity.TagCount, error)
}

type postRepository struct {
	pool *pgxpool.Pool
}

const postColumns = `id, slug, title, summary, body_markdown, COALESCE(cover_image_url, ''), tags,
	status, published_at, created_at, updated_at`

const postOrder = `ORDER BY published_at DESC NULLS FIRST, id DESC`

// postWhere filters by status and, when $2 is non-empty, by tag using the
// GIN-indexed containment operator.
const postWhere = `WHERE ($1::text IS NULL OR status = $1)
	AND ($2::text = '' OR tags @> jsonb_build_array($2::text))`

func scanPost(row pgx.Row) (entity.Post, error) {
	var p entity.Post
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Summary, &p.BodyMarkdown, &p.CoverImageURL, &p.Tags,
		&p.Status, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *postRepository) List(ctx context.Context, f entity.ListFilter) ([]entity.Post, int, error) {
	db := conn(ctx, r.pool)
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM blog_posts `+postWhere, f.Status, f.Tag).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(ctx, `SELECT `+postColumns+` FROM blog_posts `+postWhere+` `+postOrder+` LIMIT $3 OFFSET $4`,
		f.Status, f.Tag, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Post, error) { return scanPost(row) })
	return items, total, err
}

func (r *postRepository) FindByID(ctx context.Context, id int64) (entity.Post, error) {
	p, err := scanPost(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+postColumns+` FROM blog_posts WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, entity.NewNotFoundError("post not found")
	}
	return p, err
}

func (r *postRepository) FindBySlug(ctx context.Context, slug string, status *entity.Status) (entity.Post, error) {
	p, err := scanPost(conn(ctx, r.pool).QueryRow(ctx, `SELECT `+postColumns+` FROM blog_posts
		WHERE slug = $1 AND ($2::text IS NULL OR status = $2)`, slug, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, entity.NewNotFoundError("post not found")
	}
	return p, err
}

func (r *postRepository) Create(ctx context.Context, p *entity.Post) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO blog_posts (slug, title, summary, body_markdown, cover_image_url, tags, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		p.Slug, p.Title, p.Summary, p.BodyMarkdown, nullIfEmpty(p.CoverImageURL), nonNil(p.Tags), p.Status, p.PublishedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	return mapWriteError(err, "post")
}

func (r *postRepository) Update(ctx context.Context, p *entity.Post) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE blog_posts SET slug = $2, title = $3, summary = $4, body_markdown = $5,
			cover_image_url = $6, tags = $7, status = $8, published_at = $9
		WHERE id = $1
		RETURNING created_at, updated_at`,
		p.ID, p.Slug, p.Title, p.Summary, p.BodyMarkdown, nullIfEmpty(p.CoverImageURL), nonNil(p.Tags), p.Status, p.PublishedAt,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.NewNotFoundError("post not found")
	}
	return mapWriteError(err, "post")
}

func (r *postRepository) Delete(ctx context.Context, id int64) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM blog_posts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.NewNotFoundError("post not found")
	}
	return nil
}

func (r *postRepository) Tags(ctx context.Context, status *entity.Status) ([]entity.TagCount, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT tag, count(*)
		FROM blog_posts, jsonb_array_elements_text(tags) AS tag
		WHERE ($1::text IS NULL OR status = $1)
		GROUP BY tag
		ORDER BY count(*) DESC, tag`, status)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.TagCount, error) {
		var t entity.TagCount
		err := row.Scan(&t.Tag, &t.Count)
		return t, err
	})
}

func NewPostRepository(pool *pgxpool.Pool) PostRepository {
	return &postRepository{pool: pool}
}
