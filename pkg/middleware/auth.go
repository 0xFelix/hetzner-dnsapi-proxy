package middleware

import (
	"log"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func NewAuthorizer(allowedDomains config.AllowedDomains) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := reqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !CheckPermission(data.FullName, r.RemoteAddr, allowedDomains) {
				log.Printf("client '%s' is not allowed to update '%s' data of '%s' to '%s'\n",
					r.RemoteAddr, data.Type, data.FullName, data.Value)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CheckPermission(fqdn, clientIP string, allowedDomains config.AllowedDomains) bool {
	for domain, ipNets := range allowedDomains {
		if fqdn != domain && !IsSubDomain(fqdn, domain) {
			continue
		}
		for _, ipNet := range ipNets {
			ip := net.ParseIP(clientIP)
			if ip != nil && ipNet.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func IsSubDomain(sub, parent string) bool {
	subParts := strings.Split(sub, ".")
	parentParts := strings.Split(parent, ".")

	// Parent domain must be a wildcard domain
	// The subdomain must have at least the same amount of parts as the parent domain
	if parentParts[0] != "*" || len(subParts) < len(parentParts) {
		return false
	}

	// All domain parts up to the asterisk must match
	offset := len(subParts) - len(parentParts[1:])
	return slices.Equal(parentParts[1:], subParts[offset:])
}
