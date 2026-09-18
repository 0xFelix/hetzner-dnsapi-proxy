package middleware

import (
	"log"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

func NewSetClientIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addrPort, err := netip.ParseAddrPort(r.RemoteAddr)
			if err != nil {
				addr := sanitize.LogValue(r.RemoteAddr)
				//nolint:gosec // addr is sanitized above; err contains no user-controlled data
				log.Printf("failed to parse remote address %s: %v", addr, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			remote := addrPort.Addr()
			r.RemoteAddr = remote.String()
			if slices.ContainsFunc(trustedProxies, func(p netip.Prefix) bool { return p.Contains(remote) }) {
				ip := r.Header.Get("X-Real-Ip")
				if ip == "" {
					first, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
					ip = strings.TrimSpace(first)
				}
				if ip != "" {
					parsed, err := netip.ParseAddr(ip)
					if err != nil {
						sanitized := sanitize.LogValue(ip)
						//nolint:gosec // sanitized is sanitized above; r.RemoteAddr is already a parsed IP
						log.Printf("ignoring invalid forwarded client IP %q from proxy %s", sanitized, r.RemoteAddr)
					} else {
						r.RemoteAddr = parsed.String()
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
