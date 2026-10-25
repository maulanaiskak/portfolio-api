package service

import (
	"context"
	"strings"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

type MilestoneService interface {
	ListPublished(ctx context.Context, page Page) ([]entity.Milestone, int, error)
	GetPublished(ctx context.Context, slug string) (entity.Milestone, error)

	List(ctx context.Context, page Page, status *entity.Status) ([]entity.Milestone, int, error)
	Get(ctx context.Context, id int64) (entity.Milestone, error)
	Create(ctx context.Context, m entity.Milestone) (entity.Milestone, error)
	Update(ctx context.Context, m entity.Milestone) (entity.Milestone, error)
	Delete(ctx context.Context, id int64) error
}

type milestoneService struct {
	repo  repository.MilestoneRepository
	tx    repository.Transactor
	cache repository.WebCache
	now   func() time.Time
}

func (s *milestoneService) invalidate(slugs ...string) {
	tags := []string{"milestones"}
	for _, slug := range slugs {
		if slug != "" {
			tags = append(tags, "milestone:"+slug)
		}
	}
	s.cache.Invalidate(tags...)
}

func (s *milestoneService) ListPublished(ctx context.Context, page Page) ([]entity.Milestone, int, error) {
	return s.repo.List(ctx, page.filter(published()))
}

func (s *milestoneService) GetPublished(ctx context.Context, slug string) (entity.Milestone, error) {
	return s.repo.FindBySlug(ctx, slug, published())
}

func (s *milestoneService) List(ctx context.Context, page Page, status *entity.Status) ([]entity.Milestone, int, error) {
	return s.repo.List(ctx, page.filter(status))
}

func (s *milestoneService) Get(ctx context.Context, id int64) (entity.Milestone, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *milestoneService) Create(ctx context.Context, m entity.Milestone) (entity.Milestone, error) {
	if err := s.normalize(&m); err != nil {
		return m, err
	}
	if err := s.repo.Create(ctx, &m); err != nil {
		return m, err
	}
	s.invalidate(m.Slug)
	return m, nil
}

func (s *milestoneService) Update(ctx context.Context, m entity.Milestone) (entity.Milestone, error) {
	var oldSlug string
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		stored, err := s.repo.FindByID(ctx, m.ID)
		if err != nil {
			return err
		}
		oldSlug = stored.Slug
		keepImmutable(&m.Content, stored.Content)
		if err := s.normalize(&m); err != nil {
			return err
		}
		return s.repo.Update(ctx, &m)
	})
	if err == nil {
		s.invalidate(oldSlug, m.Slug)
	}
	return m, err
}

func (s *milestoneService) Delete(ctx context.Context, id int64) error {
	stored, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(stored.Slug)
	return nil
}

func (s *milestoneService) normalize(m *entity.Milestone) error {
	var v validator
	normalizeContent(&m.Content, &v, s.now())
	m.Company = strings.TrimSpace(m.Company)
	m.Role = strings.TrimSpace(m.Role)
	v.maxLen("company", m.Company, maxShortField)
	v.maxLen("role", m.Role, maxShortField)
	return v.err()
}

func NewMilestoneService(repo repository.MilestoneRepository, tx repository.Transactor, cache repository.WebCache) MilestoneService {
	return &milestoneService{repo: repo, tx: tx, cache: cache, now: time.Now}
}
