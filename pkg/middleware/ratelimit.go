package middleware

import (
	"log"
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/ratelimit"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

func NewRateLimit(limiter *ratelimit.Limiter, onExceeded http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(r.RemoteAddr) {
				addr := sanitize.LogValue(r.RemoteAddr)
				//nolint:gosec // value is sanitized above
				log.Printf("rate limit exceeded for %s", addr)
				onExceeded(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimitExceeded(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusTooManyRequests)
}

func NicRateLimitExceeded(w http.ResponseWriter, _ *http.Request) {
	writeNicToken(w, http.StatusOK, nicTokenAbuse)
}
