package middleware

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func NewShowDomainsDirectAdmin(allowedDomains config.AllowedDomains) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			domains := map[string]struct{}{}
			for domain := range allowedDomains {
				domains[strings.TrimPrefix(domain, "*.")] = struct{}{}
			}

			values := url.Values{}
			for domain := range domains {
				values.Add("list", domain)
			}

			w.Header().Set(headerContentType, applicationURLEncoded)
			if _, err := w.Write([]byte(values.Encode())); err != nil {
				log.Printf(failedWriteResponseFmt, err)
				return
			}
		})
	}
}
