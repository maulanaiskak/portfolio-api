package service

import (
	"context"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

type PostService interface {
	ListPublished(ctx context.Context, page Page, tag string) ([]entity.Post, int, error)
	GetPublished(ctx context.Context, slug string) (entity.Post, error)
	PublishedTags(ctx context.Context) ([]entity.TagCount, error)

	List(ctx context.Context, page Page, status *entity.Status) ([]entity.Post, int, error)
	Get(ctx context.Context, id int64) (entity.Post, error)
	Create(ctx context.Context, p entity.Post) (entity.Post, error)
	Update(ctx context.Context, p entity.Post) (entity.Post, error)
	Delete(ctx context.Context, id int64) error
}

type postService struct {
	repo repository.PostRepository
	tx   repository.Transactor
	now  func() time.Time
}

func (s *postService) ListPublished(ctx context.Context, page Page, tag string) ([]entity.Post, int, error) {
	f := page.filter(published())
	f.Tag = Slugify(tag)
	return s.repo.List(ctx, f)
}

func (s *postService) GetPublished(ctx context.Context, slug string) (entity.Post, error) {
	return s.repo.FindBySlug(ctx, slug, published())
}

func (s *postService) PublishedTags(ctx context.Context) ([]entity.TagCount, error) {
	return s.repo.Tags(ctx, published())
}

func (s *postService) List(ctx context.Context, page Page, status *entity.Status) ([]entity.Post, int, error) {
	return s.repo.List(ctx, page.filter(status))
}

func (s *postService) Get(ctx context.Context, id int64) (entity.Post, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *postService) Create(ctx context.Context, p entity.Post) (entity.Post, error) {
	if err := s.normalize(&p); err != nil {
		return p, err
	}
	return p, s.repo.Create(ctx, &p)
}

func (s *postService) Update(ctx context.Context, p entity.Post) (entity.Post, error) {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		stored, err := s.repo.FindByID(ctx, p.ID)
		if err != nil {
			return err
		}
		keepImmutable(&p.Content, stored.Content)
		if err := s.normalize(&p); err != nil {
			return err
		}
		return s.repo.Update(ctx, &p)
	})
	return p, err
}

func (s *postService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *postService) normalize(p *entity.Post) error {
	var v validator
	normalizeContent(&p.Content, &v, s.now())

	// Tags are stored as slugs ("Go Lang" -> "go-lang"), deduplicated.
	seen := map[string]bool{}
	tags := make([]string, 0, len(p.Tags))
	for _, raw := range p.Tags {
		t := Slugify(raw)
		if t == "" || seen[t] {
			continue
		}
		if len(t) > maxTagLen {
			v.add("tag %q must be at most %d characters", t, maxTagLen)
		}
		seen[t] = true
		tags = append(tags, t)
	}
	if len(tags) > maxTagCount {
		v.add("at most %d tags allowed", maxTagCount)
	}
	p.Tags = tags
	return v.err()
}

func NewPostService(repo repository.PostRepository, tx repository.Transactor) PostService {
	return &postService{repo: repo, tx: tx, now: time.Now}
}
