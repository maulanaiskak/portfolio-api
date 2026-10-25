package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

const (
	DefaultPageSize = 12
	MaxPageSize     = 50

	maxTitleLen    = 200
	maxSlugLen     = 100
	maxSummaryLen  = 1000
	maxBodyLen     = 200_000
	maxURLLen      = 2048
	maxTagCount    = 10
	maxTagLen      = 30
	maxShortField  = 200
	maxCaptionLen  = 500
	maxPhotosTrips = 200
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Page is a normalized page request.
type Page struct {
	Number int // 1-based
	Size   int
}

func NewPage(number, size int) Page {
	if number < 1 {
		number = 1
	}
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	// Bound offsets so deep paging can't force huge scans.
	if number > 10_000 {
		number = 10_000
	}
	return Page{Number: number, Size: size}
}

func (p Page) filter(status *entity.Status) entity.ListFilter {
	return entity.ListFilter{Status: status, Limit: p.Size, Offset: (p.Number - 1) * p.Size}
}

func published() *entity.Status {
	s := entity.StatusPublished
	return &s
}

// Slugify turns a title into a URL-safe kebab-case slug ("Japan 2024: Tokyo!"
// -> "japan-2024-tokyo"). Accents are stripped; other scripts are dropped.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFKD.String(strings.ToLower(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case unicode.Is(unicode.Mn, r):
			// combining accent from NFKD: drop
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > maxSlugLen {
		out = strings.Trim(out[:maxSlugLen], "-")
	}
	return out
}

// validator collects field problems into one validation error.
type validator struct{ problems []string }

func (v *validator) add(format string, args ...any) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func (v *validator) maxLen(field, value string, max int) {
	if len(value) > max {
		v.add("%s must be at most %d characters", field, max)
	}
}

func (v *validator) url(field, value string) {
	if value == "" {
		return
	}
	v.maxLen(field, value, maxURLLen)
	if !isHTTPURL(value) {
		v.add("%s must be an http(s) URL", field)
	}
}

func (v *validator) err() error {
	if len(v.problems) == 0 {
		return nil
	}
	return entity.NewValidationError("%s", strings.Join(v.problems, "; "))
}

// normalizeContent trims fields, fills the slug from the title, sets
// published_at on first publish, and validates the shared fields.
func normalizeContent(c *entity.Content, v *validator, now time.Time) {
	c.Title = strings.TrimSpace(c.Title)
	c.Slug = strings.TrimSpace(c.Slug)
	c.Summary = strings.TrimSpace(c.Summary)
	c.CoverImageURL = strings.TrimSpace(c.CoverImageURL)
	if c.Status == "" {
		c.Status = entity.StatusDraft
	}
	if c.Slug == "" {
		c.Slug = Slugify(c.Title)
	}

	if c.Title == "" {
		v.add("title is required")
	}
	v.maxLen("title", c.Title, maxTitleLen)
	if !slugPattern.MatchString(c.Slug) || len(c.Slug) > maxSlugLen {
		v.add("slug must be lowercase letters, digits and single dashes (max %d chars)", maxSlugLen)
	}
	v.maxLen("summary", c.Summary, maxSummaryLen)
	v.maxLen("bodyMarkdown", c.BodyMarkdown, maxBodyLen)
	v.url("coverImageUrl", c.CoverImageURL)
	if !c.Status.Valid() {
		v.add("status must be draft or published")
	}
	if c.Status == entity.StatusPublished && c.PublishedAt == nil {
		t := now.UTC()
		c.PublishedAt = &t
	}
}

// keepImmutable copies server-managed fields from the stored row onto an
// update so clients can't rewrite them.
func keepImmutable(dst *entity.Content, stored entity.Content) {
	dst.CreatedAt = stored.CreatedAt
	// Keep the original publish date across edits unless the client sets one.
	if dst.PublishedAt == nil && stored.PublishedAt != nil && dst.Status == entity.StatusPublished {
		dst.PublishedAt = stored.PublishedAt
	}
}
