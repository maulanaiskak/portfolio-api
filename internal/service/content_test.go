package service

import (
	"strings"
	"testing"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Japan 2024: Tokyo & Kyoto!": "japan-2024-tokyo-kyoto",
		"  Café  Crème  ":            "cafe-creme",
		"Why I switched majors???":   "why-i-switched-majors",
		"---":                        "",
		"日本":                         "",
		"Go/Backend":                 "go-backend",
		strings.Repeat("a", 150):     strings.Repeat("a", maxSlugLen),
		"../../etc/passwd":           "etc-passwd",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewPageClamps(t *testing.T) {
	cases := []struct{ number, size, wantNumber, wantSize int }{
		{0, 0, 1, DefaultPageSize},
		{-5, 1000, 1, MaxPageSize},
		{3, 10, 3, 10},
		{1 << 30, 10, 10_000, 10},
	}
	for _, c := range cases {
		p := NewPage(c.number, c.size)
		if p.Number != c.wantNumber || p.Size != c.wantSize {
			t.Errorf("NewPage(%d,%d) = %+v", c.number, c.size, p)
		}
	}
}

func TestNormalizeContent(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	var v validator
	c := entity.Content{Title: "  Hello World  ", Status: entity.StatusPublished}
	normalizeContent(&c, &v, now)
	if err := v.err(); err != nil {
		t.Fatal(err)
	}
	if c.Slug != "hello-world" || c.Title != "Hello World" || c.PublishedAt == nil || !c.PublishedAt.Equal(now) {
		t.Fatalf("unexpected normalized content: %+v", c)
	}

	var bad validator
	c = entity.Content{Slug: "Bad Slug", CoverImageURL: "javascript:alert(1)", Status: "live"}
	normalizeContent(&c, &bad, now)
	msg := bad.err().Error()
	for _, want := range []string{"title is required", "slug must be", "coverImageUrl must be", "status must be"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected %q in %q", want, msg)
		}
	}
}
