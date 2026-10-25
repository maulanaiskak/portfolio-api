package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type TripService interface {
	ListPublished(ctx context.Context, page Page) ([]entity.Trip, int, error)
	GetPublished(ctx context.Context, slug string) (entity.Trip, error)

	List(ctx context.Context, page Page, status *entity.Status) ([]entity.Trip, int, error)
	Get(ctx context.Context, id int64) (entity.Trip, error)
	Create(ctx context.Context, t entity.Trip) (entity.Trip, error)
	Update(ctx context.Context, t entity.Trip) (entity.Trip, error)
	Delete(ctx context.Context, id int64) error

	AddPhoto(ctx context.Context, p entity.TripPhoto) (entity.TripPhoto, error)
	UpdatePhoto(ctx context.Context, p entity.TripPhoto) (entity.TripPhoto, error)
	DeletePhoto(ctx context.Context, tripID, photoID int64) error
}

type tripService struct {
	repo repository.TripRepository
	tx   repository.Transactor
	now  func() time.Time
}

func (s *tripService) ListPublished(ctx context.Context, page Page) ([]entity.Trip, int, error) {
	return s.repo.List(ctx, page.filter(published()))
}

func (s *tripService) GetPublished(ctx context.Context, slug string) (entity.Trip, error) {
	t, err := s.repo.FindBySlug(ctx, slug, published())
	if err != nil {
		return t, err
	}
	t.Photos, err = s.repo.ListPhotos(ctx, t.ID)
	return t, err
}

func (s *tripService) List(ctx context.Context, page Page, status *entity.Status) ([]entity.Trip, int, error) {
	return s.repo.List(ctx, page.filter(status))
}

func (s *tripService) Get(ctx context.Context, id int64) (entity.Trip, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return t, err
	}
	t.Photos, err = s.repo.ListPhotos(ctx, t.ID)
	return t, err
}

func (s *tripService) Create(ctx context.Context, t entity.Trip) (entity.Trip, error) {
	if err := s.normalize(&t); err != nil {
		return t, err
	}
	if err := s.repo.Create(ctx, &t); err != nil {
		return t, err
	}
	t.Photos = []entity.TripPhoto{}
	return t, nil
}

func (s *tripService) Update(ctx context.Context, t entity.Trip) (entity.Trip, error) {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		stored, err := s.repo.FindByID(ctx, t.ID)
		if err != nil {
			return err
		}
		keepImmutable(&t.Content, stored.Content)
		if err := s.normalize(&t); err != nil {
			return err
		}
		if err := s.repo.Update(ctx, &t); err != nil {
			return err
		}
		t.Photos, err = s.repo.ListPhotos(ctx, t.ID)
		return err
	})
	return t, err
}

func (s *tripService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *tripService) AddPhoto(ctx context.Context, p entity.TripPhoto) (entity.TripPhoto, error) {
	if err := validatePhoto(&p, true); err != nil {
		return p, err
	}
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.FindByID(ctx, p.TripID); err != nil {
			return err
		}
		photos, err := s.repo.ListPhotos(ctx, p.TripID)
		if err != nil {
			return err
		}
		if len(photos) >= maxPhotosTrips {
			return entity.NewValidationError("a trip can have at most %d photos", maxPhotosTrips)
		}
		// New photos go to the end unless a position was given.
		if p.SortOrder == 0 && len(photos) > 0 {
			p.SortOrder = photos[len(photos)-1].SortOrder + 1
		}
		return s.repo.CreatePhoto(ctx, &p)
	})
	return p, err
}

func (s *tripService) UpdatePhoto(ctx context.Context, p entity.TripPhoto) (entity.TripPhoto, error) {
	if err := validatePhoto(&p, false); err != nil {
		return p, err
	}
	return p, s.repo.UpdatePhoto(ctx, &p)
}

func (s *tripService) DeletePhoto(ctx context.Context, tripID, photoID int64) error {
	return s.repo.DeletePhoto(ctx, tripID, photoID)
}

func (s *tripService) normalize(t *entity.Trip) error {
	var v validator
	normalizeContent(&t.Content, &v, s.now())
	t.Location = strings.TrimSpace(t.Location)
	t.CountryCode = strings.ToUpper(strings.TrimSpace(t.CountryCode))
	v.maxLen("location", t.Location, maxShortField)
	if t.CountryCode != "" && !countryCodePattern.MatchString(t.CountryCode) {
		v.add("countryCode must be a 2-letter ISO code")
	}
	if t.StartDate != nil && t.EndDate != nil && t.EndDate.Before(*t.StartDate) {
		v.add("endDate must be on or after startDate")
	}
	return v.err()
}

func validatePhoto(p *entity.TripPhoto, requireURL bool) error {
	var v validator
	p.URL = strings.TrimSpace(p.URL)
	p.Caption = strings.TrimSpace(p.Caption)
	if requireURL {
		if p.URL == "" {
			v.add("url is required")
		}
		v.url("url", p.URL)
	}
	v.maxLen("caption", p.Caption, maxCaptionLen)
	if p.SortOrder < 0 {
		v.add("sortOrder must not be negative")
	}
	return v.err()
}

func NewTripService(repo repository.TripRepository, tx repository.Transactor) TripService {
	return &tripService{repo: repo, tx: tx, now: time.Now}
}
