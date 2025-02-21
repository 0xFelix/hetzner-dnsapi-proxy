package middleware

import (
	"log"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func NewAuthorizer(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := reqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !CheckPermission(cfg, data, r.RemoteAddr) {
				log.Printf("client '%s' is not allowed to update '%s' data of '%s' to '%s'",
					r.RemoteAddr, data.Type, data.FullName, data.Value)
				if cfg.Auth.Method != config.AuthMethodAllowedDomains && data.BasicAuth {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CheckPermission(cfg *config.Config, data *ReqData, remoteAddr string) bool {
	if !config.AuthMethodIsValid(cfg.Auth.Method) {
		log.Printf("invalid auth method: %s", cfg.Auth.Method)
		return false
	}

	allowedAllowedDomains := CheckAllowedDomains(data.FullName, remoteAddr, cfg.Auth.AllowedDomains)
	if cfg.Auth.Method == config.AuthMethodAllowedDomains {
		return allowedAllowedDomains
	}

	allowedUsers := CheckUsers(data.FullName, data.Username, data.Password, cfg.Auth.Users)
	if cfg.Auth.Method == config.AuthMethodUsers {
		return allowedUsers
	}

	if cfg.Auth.Method == config.AuthMethodBoth {
		return allowedAllowedDomains && allowedUsers
	}

	if cfg.Auth.Method == config.AuthMethodAny {
		return allowedAllowedDomains || allowedUsers
	}

	return false
}

func CheckAllowedDomains(fqdn, clientIP string, allowedDomains config.AllowedDomains) bool {
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

func CheckUsers(fqdn, username, password string, users []config.User) bool {
	if fqdn == "" || username == "" || password == "" {
		return false
	}
	for _, user := range users {
		if user.Username == username && user.Password == password {
			for _, domain := range user.Domains {
				if fqdn == domain || IsSubDomain(fqdn, domain) {
					return true
				}
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
