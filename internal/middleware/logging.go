package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger writes one structured line per request. It logs only method,
// path, status, size, duration, request id and client IP: never headers,
// query strings or bodies, which can carry tokens or credentials.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		level := slog.LevelInfo
		if ww.Status() >= 500 {
			level = slog.LevelError
		}
		slog.LogAttrs(r.Context(), level, "http request",
			slog.String("request_id", chimw.GetReqID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote_ip", r.RemoteAddr),
		)
	})
}
