package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type CvSkillRepository interface {
	// FindAll returns categories in order, each with its items in order.
	FindAll(ctx context.Context) ([]entity.SkillCategory, error)
	CreateCategory(ctx context.Context, c *entity.SkillCategory) error
	CreateItem(ctx context.Context, categoryID int64, item *entity.SkillItem) error
	// DeleteAll removes all categories; items go with them via ON DELETE CASCADE.
	DeleteAll(ctx context.Context) error
	CountCategories(ctx context.Context) (int, error)
	CountItems(ctx context.Context) (int, error)
}

type cvSkillRepository struct {
	pool *pgxpool.Pool
}

func (r *cvSkillRepository) FindAll(ctx context.Context) ([]entity.SkillCategory, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, `
		SELECT c.id, c.category, c.sort_order,
		       i.id, i.name, i.level, i.note, i.link, i.sort_order
		FROM cv_skill_categories c
		LEFT JOIN cv_skill_items i ON i.category_id = c.id
		ORDER BY c.sort_order, c.id, i.sort_order, i.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []entity.SkillCategory{}
	for rows.Next() {
		var (
			cat                    entity.SkillCategory
			itemID, level, itemOrd *int64
			name                   *string
			item                   entity.SkillItem
		)
		if err := rows.Scan(&cat.ID, &cat.Category, &cat.SortOrder,
			&itemID, &name, &level, &item.Note, &item.Link, &itemOrd); err != nil {
			return nil, err
		}
		if n := len(categories); n == 0 || categories[n-1].ID != cat.ID {
			cat.Items = []entity.SkillItem{}
			categories = append(categories, cat)
		}
		if itemID == nil {
			continue // category with no items (LEFT JOIN)
		}
		item.ID, item.Name, item.Level, item.SortOrder = *itemID, *name, int(*level), int(*itemOrd)
		last := &categories[len(categories)-1]
		last.Items = append(last.Items, item)
	}
	return categories, rows.Err()
}

func (r *cvSkillRepository) CreateCategory(ctx context.Context, c *entity.SkillCategory) error {
	return conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cv_skill_categories (category, sort_order)
		VALUES ($1, $2)
		RETURNING id`,
		c.Category, c.SortOrder,
	).Scan(&c.ID)
}

func (r *cvSkillRepository) CreateItem(ctx context.Context, categoryID int64, item *entity.SkillItem) error {
	return conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cv_skill_items (category_id, name, level, note, link, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		categoryID, item.Name, item.Level, item.Note, item.Link, item.SortOrder,
	).Scan(&item.ID)
}

func (r *cvSkillRepository) DeleteAll(ctx context.Context) error {
	_, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM cv_skill_categories`)
	return err
}

func (r *cvSkillRepository) CountCategories(ctx context.Context) (int, error) {
	var n int
	err := conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM cv_skill_categories`).Scan(&n)
	return n, err
}

func (r *cvSkillRepository) CountItems(ctx context.Context) (int, error) {
	var n int
	err := conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM cv_skill_items`).Scan(&n)
	return n, err
}

func NewCvSkillRepository(pool *pgxpool.Pool) CvSkillRepository {
	return &cvSkillRepository{pool: pool}
}
