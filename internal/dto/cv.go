package dto

import "github.com/maulanaiskak/portfolio-api/internal/entity"

// JSON field names intentionally mirror the TS interfaces in the current
// frontend (maulanaiskak-nextjs/src/types/index.ts) so the web app can switch
// data sources without reshaping anything.

type ExperienceResponse struct {
	ID                      int64                   `json:"id"`
	Company                 string                  `json:"company"`
	Location                string                  `json:"location"`
	Position                string                  `json:"position"`
	Period                  string                  `json:"period"`
	Responsibilities        []entity.Responsibility `json:"responsibilities"`
	ProjectsAndAchievements []string                `json:"projectsAndAchievements"`
	Tech                    []string                `json:"tech"`
}

type ProjectResponse struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Image        string   `json:"image"`
	Highlights   []string `json:"highlights"`
}

type EducationResponse struct {
	ID          int64  `json:"id"`
	Institution string `json:"institution"`
	Location    string `json:"location"`
	Degree      string `json:"degree"`
	Period      string `json:"period"`
	GPA         string `json:"gpa"`
	Thesis      string `json:"thesis"`
	ThesisLink  string `json:"thesis_link"`
}

type SkillItemResponse struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Level int     `json:"level"`
	Note  *string `json:"note,omitempty"`
	Link  *string `json:"link,omitempty"`
}

type SkillCategoryResponse struct {
	ID       int64               `json:"id"`
	Category string              `json:"category"`
	Items    []SkillItemResponse `json:"items"`
}

// CvResponse is the payload of GET /api/v1/cv.
type CvResponse struct {
	Experiences []ExperienceResponse    `json:"experiences"`
	Projects    []ProjectResponse       `json:"projects"`
	Education   []EducationResponse     `json:"education"`
	Skills      []SkillCategoryResponse `json:"skills"`
}

func NewCvResponse(cv entity.Cv) CvResponse {
	resp := CvResponse{
		Experiences: make([]ExperienceResponse, 0, len(cv.Experiences)),
		Projects:    make([]ProjectResponse, 0, len(cv.Projects)),
		Education:   make([]EducationResponse, 0, len(cv.Education)),
		Skills:      make([]SkillCategoryResponse, 0, len(cv.Skills)),
	}
	for _, e := range cv.Experiences {
		resp.Experiences = append(resp.Experiences, ExperienceResponse{
			ID: e.ID, Company: e.Company, Location: e.Location, Position: e.Position, Period: e.Period,
			Responsibilities:        orEmpty(e.Responsibilities),
			ProjectsAndAchievements: orEmpty(e.ProjectsAndAchievements),
			Tech:                    orEmpty(e.Tech),
		})
	}
	for _, p := range cv.Projects {
		resp.Projects = append(resp.Projects, ProjectResponse{
			ID: p.ID, Title: p.Title, Description: p.Description, Image: p.Image,
			Technologies: orEmpty(p.Technologies),
			Highlights:   orEmpty(p.Highlights),
		})
	}
	for _, e := range cv.Education {
		resp.Education = append(resp.Education, EducationResponse{
			ID: e.ID, Institution: e.Institution, Location: e.Location, Degree: e.Degree,
			Period: e.Period, GPA: e.GPA, Thesis: e.Thesis, ThesisLink: e.ThesisLink,
		})
	}
	for _, c := range cv.Skills {
		cat := SkillCategoryResponse{ID: c.ID, Category: c.Category, Items: make([]SkillItemResponse, 0, len(c.Items))}
		for _, i := range c.Items {
			cat.Items = append(cat.Items, SkillItemResponse{ID: i.ID, Name: i.Name, Level: i.Level, Note: i.Note, Link: i.Link})
		}
		resp.Skills = append(resp.Skills, cat)
	}
	return resp
}

// CvRequest replaces the whole CV (seed import and admin PUT /cv). Keys
// match CvResponse so the admin editor can round-trip it; ids are accepted
// and ignored. Array order becomes sort_order.
type CvRequest struct {
	Experiences []ExperienceInput    `json:"experiences"`
	Projects    []ProjectInput       `json:"projects"`
	Education   []EducationInput     `json:"education"`
	Skills      []SkillCategoryInput `json:"skills"`
}

type ExperienceInput struct {
	ID                      int64                   `json:"id,omitempty"` // ignored
	Company                 string                  `json:"company"`
	Location                string                  `json:"location"`
	Position                string                  `json:"position"`
	Period                  string                  `json:"period"`
	Responsibilities        []entity.Responsibility `json:"responsibilities"`
	ProjectsAndAchievements []string                `json:"projectsAndAchievements"`
	Tech                    []string                `json:"tech"`
}

type ProjectInput struct {
	ID           int64    `json:"id,omitempty"` // ignored
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Image        string   `json:"image"`
	Highlights   []string `json:"highlights"`
}

type EducationInput struct {
	ID          int64  `json:"id,omitempty"` // ignored
	Institution string `json:"institution"`
	Location    string `json:"location"`
	Degree      string `json:"degree"`
	Period      string `json:"period"`
	GPA         string `json:"gpa"`
	Thesis      string `json:"thesis"`
	ThesisLink  string `json:"thesis_link"`
}

type SkillItemInput struct {
	ID    int64   `json:"id,omitempty"` // ignored
	Name  string  `json:"name"`
	Level int     `json:"level"`
	Note  *string `json:"note,omitempty"`
	Link  *string `json:"link,omitempty"`
}

type SkillCategoryInput struct {
	ID       int64            `json:"id,omitempty"` // ignored
	Category string           `json:"category"`
	Items    []SkillItemInput `json:"items"`
}

func (r CvRequest) ToEntity() entity.Cv {
	var cv entity.Cv
	for i, e := range r.Experiences {
		cv.Experiences = append(cv.Experiences, entity.Experience{
			Company: e.Company, Location: e.Location, Position: e.Position, Period: e.Period,
			Responsibilities: e.Responsibilities, ProjectsAndAchievements: e.ProjectsAndAchievements,
			Tech: e.Tech, SortOrder: i,
		})
	}
	for i, p := range r.Projects {
		cv.Projects = append(cv.Projects, entity.Project{
			Title: p.Title, Description: p.Description, Technologies: p.Technologies,
			Image: p.Image, Highlights: p.Highlights, SortOrder: i,
		})
	}
	for i, e := range r.Education {
		cv.Education = append(cv.Education, entity.Education{
			Institution: e.Institution, Location: e.Location, Degree: e.Degree, Period: e.Period,
			GPA: e.GPA, Thesis: e.Thesis, ThesisLink: e.ThesisLink, SortOrder: i,
		})
	}
	for i, c := range r.Skills {
		cat := entity.SkillCategory{Category: c.Category, SortOrder: i}
		for j, item := range c.Items {
			cat.Items = append(cat.Items, entity.SkillItem{
				Name: item.Name, Level: item.Level, Note: item.Note, Link: item.Link, SortOrder: j,
			})
		}
		cv.Skills = append(cv.Skills, cat)
	}
	return cv
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
