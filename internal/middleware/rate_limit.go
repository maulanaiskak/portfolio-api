package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/maulanaiskak/portfolio-api/internal/response"
)

// RateLimitByIP applies a per-client-IP sliding-window limit. Client IP comes
// from RemoteAddr, which chi's RealIP only rewrites when TRUST_PROXY is on,
// so clients can't dodge the limit by spoofing X-Forwarded-For.
func RateLimitByIP(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.Limit(
		requestsPerMinute,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			response.Error(w, r, response.ErrTooManyRequests)
		}),
	)
}
