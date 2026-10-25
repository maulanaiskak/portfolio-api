package dto

import (
	"strings"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

const dateLayout = "2006-01-02"

// parseDate parses an optional YYYY-MM-DD date; "" or nil means not set.
func parseDate(field string, s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, strings.TrimSpace(*s))
	if err != nil {
		return nil, entity.NewValidationError("%s must be a date in YYYY-MM-DD format", field)
	}
	return &t, nil
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

// ContentRequest holds fields shared by trip/milestone/post writes.
type ContentRequest struct {
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	BodyMarkdown  string     `json:"bodyMarkdown"`
	CoverImageURL string     `json:"coverImageUrl"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"publishedAt"`
}

func (r ContentRequest) toEntity() entity.Content {
	return entity.Content{
		Slug: r.Slug, Title: r.Title, Summary: r.Summary, BodyMarkdown: r.BodyMarkdown,
		CoverImageURL: r.CoverImageURL, Status: entity.Status(r.Status), PublishedAt: r.PublishedAt,
	}
}

// ContentSummary is the list-view shape (no markdown body).
type ContentSummary struct {
	ID            int64      `json:"id"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	CoverImageURL string     `json:"coverImageUrl"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"publishedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func newContentSummary(c entity.Content) ContentSummary {
	var published *time.Time
	if c.PublishedAt != nil {
		t := c.PublishedAt.UTC()
		published = &t
	}
	return ContentSummary{
		ID: c.ID, Slug: c.Slug, Title: c.Title, Summary: c.Summary, CoverImageURL: c.CoverImageURL,
		Status: string(c.Status), PublishedAt: published, CreatedAt: c.CreatedAt.UTC(), UpdatedAt: c.UpdatedAt.UTC(),
	}
}

// --- Trips -------------------------------------------------------------------

type TripRequest struct {
	ContentRequest
	Location    string  `json:"location"`
	CountryCode string  `json:"countryCode"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

func (r TripRequest) ToEntity() (entity.Trip, error) {
	start, err := parseDate("startDate", r.StartDate)
	if err != nil {
		return entity.Trip{}, err
	}
	end, err := parseDate("endDate", r.EndDate)
	if err != nil {
		return entity.Trip{}, err
	}
	return entity.Trip{
		Content: r.ContentRequest.toEntity(), Location: r.Location, CountryCode: r.CountryCode,
		StartDate: start, EndDate: end,
	}, nil
}

type TripSummaryResponse struct {
	ContentSummary
	Location    string  `json:"location"`
	CountryCode string  `json:"countryCode"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

type TripPhotoResponse struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sortOrder"`
}

type TripResponse struct {
	TripSummaryResponse
	BodyMarkdown string              `json:"bodyMarkdown"`
	Photos       []TripPhotoResponse `json:"photos"`
}

func NewTripSummary(t entity.Trip) TripSummaryResponse {
	return TripSummaryResponse{
		ContentSummary: newContentSummary(t.Content), Location: t.Location, CountryCode: t.CountryCode,
		StartDate: formatDate(t.StartDate), EndDate: formatDate(t.EndDate),
	}
}

func NewTripSummaries(trips []entity.Trip) []TripSummaryResponse {
	out := make([]TripSummaryResponse, 0, len(trips))
	for _, t := range trips {
		out = append(out, NewTripSummary(t))
	}
	return out
}

func NewTripResponse(t entity.Trip) TripResponse {
	photos := make([]TripPhotoResponse, 0, len(t.Photos))
	for _, p := range t.Photos {
		photos = append(photos, NewTripPhotoResponse(p))
	}
	return TripResponse{TripSummaryResponse: NewTripSummary(t), BodyMarkdown: t.BodyMarkdown, Photos: photos}
}

type TripPhotoRequest struct {
	URL       string `json:"url"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sortOrder"`
}

func NewTripPhotoResponse(p entity.TripPhoto) TripPhotoResponse {
	return TripPhotoResponse{ID: p.ID, URL: p.URL, Caption: p.Caption, SortOrder: p.SortOrder}
}

// --- Career milestones -------------------------------------------------------

type MilestoneRequest struct {
	ContentRequest
	Company       string  `json:"company"`
	Role          string  `json:"role"`
	MilestoneDate *string `json:"milestoneDate"`
}

func (r MilestoneRequest) ToEntity() (entity.Milestone, error) {
	date, err := parseDate("milestoneDate", r.MilestoneDate)
	if err != nil {
		return entity.Milestone{}, err
	}
	return entity.Milestone{Content: r.ContentRequest.toEntity(), Company: r.Company, Role: r.Role, MilestoneDate: date}, nil
}

type MilestoneSummaryResponse struct {
	ContentSummary
	Company       string  `json:"company"`
	Role          string  `json:"role"`
	MilestoneDate *string `json:"milestoneDate"`
}

type MilestoneResponse struct {
	MilestoneSummaryResponse
	BodyMarkdown string `json:"bodyMarkdown"`
}

func NewMilestoneSummary(m entity.Milestone) MilestoneSummaryResponse {
	return MilestoneSummaryResponse{
		ContentSummary: newContentSummary(m.Content), Company: m.Company, Role: m.Role,
		MilestoneDate: formatDate(m.MilestoneDate),
	}
}

func NewMilestoneSummaries(items []entity.Milestone) []MilestoneSummaryResponse {
	out := make([]MilestoneSummaryResponse, 0, len(items))
	for _, m := range items {
		out = append(out, NewMilestoneSummary(m))
	}
	return out
}

func NewMilestoneResponse(m entity.Milestone) MilestoneResponse {
	return MilestoneResponse{MilestoneSummaryResponse: NewMilestoneSummary(m), BodyMarkdown: m.BodyMarkdown}
}

// --- Blog posts --------------------------------------------------------------

type PostRequest struct {
	ContentRequest
	Tags []string `json:"tags"`
}

func (r PostRequest) ToEntity() entity.Post {
	return entity.Post{Content: r.ContentRequest.toEntity(), Tags: r.Tags}
}

type PostSummaryResponse struct {
	ContentSummary
	Tags        []string `json:"tags"`
	ReadMinutes int      `json:"readMinutes"`
}

type PostResponse struct {
	PostSummaryResponse
	BodyMarkdown string `json:"bodyMarkdown"`
}

// readMinutes estimates reading time at ~220 words per minute.
func readMinutes(markdown string) int {
	words := len(strings.Fields(markdown))
	if m := (words + 219) / 220; m > 1 {
		return m
	}
	return 1
}

func NewPostSummary(p entity.Post) PostSummaryResponse {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return PostSummaryResponse{ContentSummary: newContentSummary(p.Content), Tags: tags, ReadMinutes: readMinutes(p.BodyMarkdown)}
}

func NewPostSummaries(items []entity.Post) []PostSummaryResponse {
	out := make([]PostSummaryResponse, 0, len(items))
	for _, p := range items {
		out = append(out, NewPostSummary(p))
	}
	return out
}

func NewPostResponse(p entity.Post) PostResponse {
	return PostResponse{PostSummaryResponse: NewPostSummary(p), BodyMarkdown: p.BodyMarkdown}
}

type TagCountResponse struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func NewTagCounts(tags []entity.TagCount) []TagCountResponse {
	out := make([]TagCountResponse, 0, len(tags))
	for _, t := range tags {
		out = append(out, TagCountResponse{Tag: t.Tag, Count: t.Count})
	}
	return out
}
