package dns

import (
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func New(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// noop
			next.ServeHTTP(w, r)
		})
	}
}
