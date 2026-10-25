package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// WebCache invalidates cached pages on the web frontend (Next.js ISR tags)
// after content changes, so published edits appear immediately.
type WebCache interface {
	// Invalidate is fire-and-forget: failures are logged, never returned,
	// because the ISR window (60s) still guarantees eventual freshness.
	Invalidate(tags ...string)
}

type noopWebCache struct{}

func (noopWebCache) Invalidate(...string) {}

type nextWebCache struct {
	url    string
	secret string
	client *http.Client
}

func (c *nextWebCache) Invalidate(tags ...string) {
	if len(tags) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.send(ctx, tags); err != nil {
			slog.Warn("web cache invalidation failed", "err", err, "tags", tags)
		}
	}()
}

func (c *nextWebCache) send(ctx context.Context, tags []string) error {
	body, _ := json.Marshal(map[string][]string{"tags": tags})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Revalidate-Secret", c.secret)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("revalidate returned %d", res.StatusCode)
	}
	return nil
}

// NewWebCache returns a no-op cache when url or secret is empty.
func NewWebCache(url, secret string) WebCache {
	if url == "" || secret == "" {
		return noopWebCache{}
	}
	return &nextWebCache{url: url, secret: secret, client: &http.Client{Timeout: 5 * time.Second}}
}
