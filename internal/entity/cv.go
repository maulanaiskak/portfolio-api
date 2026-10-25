package entity

type Responsibility struct {
	Period      string `json:"period"`
	Description string `json:"description"`
}

type Experience struct {
	ID                      int64
	Company                 string
	Location                string
	Position                string
	Period                  string
	Responsibilities        []Responsibility // stored as JSONB
	ProjectsAndAchievements []string         // stored as JSONB
	Tech                    []string         // stored as JSONB
	SortOrder               int
}

type Project struct {
	ID           int64
	Title        string
	Description  string
	Technologies []string // stored as JSONB
	Image        string
	Highlights   []string // stored as JSONB
	SortOrder    int
}

type Education struct {
	ID          int64
	Institution string
	Location    string
	Degree      string
	Period      string
	GPA         string
	Thesis      string
	ThesisLink  string
	SortOrder   int
}

type SkillItem struct {
	ID        int64
	Name      string
	Level     int // 0-100
	Note      *string
	Link      *string
	SortOrder int
}

type SkillCategory struct {
	ID        int64
	Category  string
	Items     []SkillItem
	SortOrder int
}

// Cv is the whole CV, used both for the public aggregate read and as a
// snapshot for bulk import.
type Cv struct {
	Experiences []Experience
	Projects    []Project
	Education   []Education
	Skills      []SkillCategory
}

type CvCounts struct {
	Experiences     int `json:"experiences"`
	Projects        int `json:"projects"`
	Education       int `json:"education"`
	SkillCategories int `json:"skillCategories"`
	SkillItems      int `json:"skillItems"`
}

func (c Cv) Counts() CvCounts {
	counts := CvCounts{
		Experiences:     len(c.Experiences),
		Projects:        len(c.Projects),
		Education:       len(c.Education),
		SkillCategories: len(c.Skills),
	}
	for _, cat := range c.Skills {
		counts.SkillItems += len(cat.Items)
	}
	return counts
}
