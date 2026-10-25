package entity

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

func (s Status) Valid() bool { return s == StatusDraft || s == StatusPublished }

// Content holds the fields shared by trips, career milestones and blog
// posts. Empty strings mean "not set" (stored as NULL where nullable).
type Content struct {
	ID            int64
	Slug          string
	Title         string
	Summary       string
	BodyMarkdown  string
	CoverImageURL string
	Status        Status
	PublishedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Trip struct {
	Content
	Location    string
	CountryCode string // ISO 3166-1 alpha-2, uppercase
	StartDate   *time.Time
	EndDate     *time.Time
	Photos      []TripPhoto
}

type TripPhoto struct {
	ID        int64
	TripID    int64
	URL       string
	Caption   string
	SortOrder int
	CreatedAt time.Time
}

type Milestone struct {
	Content
	Company       string
	Role          string
	MilestoneDate *time.Time
}

type Post struct {
	Content
	Tags []string
}

// ListFilter selects a page of content. A nil Status means any status
// (admin); public callers always pass published.
type ListFilter struct {
	Status *Status
	Tag    string // posts only
	Limit  int
	Offset int
}

type TagCount struct {
	Tag   string
	Count int
}
