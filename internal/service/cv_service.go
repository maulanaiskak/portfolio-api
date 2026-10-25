package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

type CvService interface {
	GetCv(ctx context.Context) (entity.Cv, error)
	// ImportCv loads a full CV snapshot in one transaction. With replace=false
	// it refuses to run when any CV table already has rows. It returns the row
	// counts read back from the DB, and rolls back if they don't match input.
	ImportCv(ctx context.Context, cv entity.Cv, replace bool) (entity.CvCounts, error)
}

type cvService struct {
	experienceRepo repository.CvExperienceRepository
	projectRepo    repository.CvProjectRepository
	educationRepo  repository.CvEducationRepository
	skillRepo      repository.CvSkillRepository
	tx             repository.Transactor
}

func (s *cvService) GetCv(ctx context.Context) (entity.Cv, error) {
	var (
		cv  entity.Cv
		err error
	)
	if cv.Experiences, err = s.experienceRepo.FindAll(ctx); err != nil {
		return cv, fmt.Errorf("list experiences: %w", err)
	}
	if cv.Projects, err = s.projectRepo.FindAll(ctx); err != nil {
		return cv, fmt.Errorf("list projects: %w", err)
	}
	if cv.Education, err = s.educationRepo.FindAll(ctx); err != nil {
		return cv, fmt.Errorf("list education: %w", err)
	}
	if cv.Skills, err = s.skillRepo.FindAll(ctx); err != nil {
		return cv, fmt.Errorf("list skills: %w", err)
	}
	return cv, nil
}

func (s *cvService) ImportCv(ctx context.Context, cv entity.Cv, replace bool) (entity.CvCounts, error) {
	if err := validateCv(cv); err != nil {
		return entity.CvCounts{}, err
	}

	var counts entity.CvCounts
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		existing, err := s.counts(ctx)
		if err != nil {
			return err
		}
		if existing != (entity.CvCounts{}) {
			if !replace {
				return entity.NewConflictError("CV tables are not empty (%+v); rerun with replace to overwrite", existing)
			}
			if err := s.deleteAll(ctx); err != nil {
				return err
			}
		}

		for i := range cv.Experiences {
			if err := s.experienceRepo.Create(ctx, &cv.Experiences[i]); err != nil {
				return fmt.Errorf("insert experience %q: %w", cv.Experiences[i].Company, err)
			}
		}
		for i := range cv.Projects {
			if err := s.projectRepo.Create(ctx, &cv.Projects[i]); err != nil {
				return fmt.Errorf("insert project %q: %w", cv.Projects[i].Title, err)
			}
		}
		for i := range cv.Education {
			if err := s.educationRepo.Create(ctx, &cv.Education[i]); err != nil {
				return fmt.Errorf("insert education %q: %w", cv.Education[i].Institution, err)
			}
		}
		for i := range cv.Skills {
			cat := &cv.Skills[i]
			if err := s.skillRepo.CreateCategory(ctx, cat); err != nil {
				return fmt.Errorf("insert skill category %q: %w", cat.Category, err)
			}
			for j := range cat.Items {
				if err := s.skillRepo.CreateItem(ctx, cat.ID, &cat.Items[j]); err != nil {
					return fmt.Errorf("insert skill %q: %w", cat.Items[j].Name, err)
				}
			}
		}

		// Verify inside the tx so a mismatch rolls everything back.
		if counts, err = s.counts(ctx); err != nil {
			return err
		}
		if want := cv.Counts(); counts != want {
			return fmt.Errorf("row count mismatch after import: want %+v, got %+v", want, counts)
		}
		return nil
	})
	return counts, err
}

func (s *cvService) counts(ctx context.Context) (entity.CvCounts, error) {
	var (
		c   entity.CvCounts
		err error
	)
	if c.Experiences, err = s.experienceRepo.Count(ctx); err != nil {
		return c, err
	}
	if c.Projects, err = s.projectRepo.Count(ctx); err != nil {
		return c, err
	}
	if c.Education, err = s.educationRepo.Count(ctx); err != nil {
		return c, err
	}
	if c.SkillCategories, err = s.skillRepo.CountCategories(ctx); err != nil {
		return c, err
	}
	if c.SkillItems, err = s.skillRepo.CountItems(ctx); err != nil {
		return c, err
	}
	return c, nil
}

func (s *cvService) deleteAll(ctx context.Context) error {
	if err := s.experienceRepo.DeleteAll(ctx); err != nil {
		return err
	}
	if err := s.projectRepo.DeleteAll(ctx); err != nil {
		return err
	}
	if err := s.educationRepo.DeleteAll(ctx); err != nil {
		return err
	}
	return s.skillRepo.DeleteAll(ctx)
}

func validateCv(cv entity.Cv) error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	for i, e := range cv.Experiences {
		if strings.TrimSpace(e.Company) == "" || strings.TrimSpace(e.Position) == "" {
			add("experience[%d]: company and position are required", i)
		}
	}
	for i, p := range cv.Projects {
		if strings.TrimSpace(p.Title) == "" {
			add("projects[%d]: title is required", i)
		}
	}
	for i, e := range cv.Education {
		if strings.TrimSpace(e.Institution) == "" {
			add("education[%d]: institution is required", i)
		}
		if e.ThesisLink != "" && !isHTTPURL(e.ThesisLink) {
			add("education[%d]: thesis_link must be an http(s) URL", i)
		}
	}
	seen := map[string]bool{}
	for i, c := range cv.Skills {
		if strings.TrimSpace(c.Category) == "" {
			add("skills[%d]: category is required", i)
		}
		if seen[c.Category] {
			add("skills[%d]: duplicate category %q", i, c.Category)
		}
		seen[c.Category] = true
		for j, item := range c.Items {
			if strings.TrimSpace(item.Name) == "" {
				add("skills[%d].items[%d]: name is required", i, j)
			}
			if item.Level < 0 || item.Level > 100 {
				add("skills[%d].items[%d]: level must be 0-100", i, j)
			}
			if item.Link != nil && !isHTTPURL(*item.Link) {
				add("skills[%d].items[%d]: link must be an http(s) URL", i, j)
			}
		}
	}
	if len(problems) > 0 {
		return entity.NewValidationError("%s", strings.Join(problems, "; "))
	}
	return nil
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func NewCvService(
	experienceRepo repository.CvExperienceRepository,
	projectRepo repository.CvProjectRepository,
	educationRepo repository.CvEducationRepository,
	skillRepo repository.CvSkillRepository,
	tx repository.Transactor,
) CvService {
	return &cvService{
		experienceRepo: experienceRepo,
		projectRepo:    projectRepo,
		educationRepo:  educationRepo,
		skillRepo:      skillRepo,
		tx:             tx,
	}
}
