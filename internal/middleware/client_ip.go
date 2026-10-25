package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP sets r.RemoteAddr from a single header that the trusted edge
// proxy overwrites on every request (Fly.io: Fly-Client-IP). Unlike
// X-Forwarded-For, whose first entry is client-controlled, such a header
// can't be spoofed to dodge per-IP rate limits. Use only behind that proxy.
func ClientIP(header string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ip := strings.TrimSpace(r.Header.Get(header)); net.ParseIP(ip) != nil {
				r.RemoteAddr = net.JoinHostPort(ip, "0")
			}
			next.ServeHTTP(w, r)
		})
	}
}
