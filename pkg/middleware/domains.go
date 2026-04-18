package middleware

import (
	"log"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/ratelimit"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

func NewShowDomainsDirectAdmin(cfg *config.Config, lockout *ratelimit.Lockout) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.AuthMethodIsValid(cfg.Auth.Method) {
				log.Printf("invalid auth method: %s", cfg.Auth.Method)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if lockout.IsBlocked(r.RemoteAddr) {
				logLockedOut(r.RemoteAddr)
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			username, password, _ := r.BasicAuth()
			usesUsers := authMethodUsesUsers(cfg.Auth.Method)
			if usesUsers && (username != "" || password != "") {
				if checkUserCredentials(username, password, cfg.Auth.Users) {
					lockout.Reset(r.RemoteAddr)
				} else {
					lockout.RecordFailure(r.RemoteAddr)
				}
			}

			domains := GetDomains(cfg, r.RemoteAddr, username, password)
			if len(domains) == 0 {
				addr := sanitize.LogValue(r.RemoteAddr)
				//nolint:gosec // value is sanitized above
				log.Printf("client '%s' is not allowed to list any domains", addr)
				if usesUsers {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
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

func authMethodUsesUsers(method string) bool {
	return method == config.AuthMethodUsers ||
		method == config.AuthMethodBoth ||
		method == config.AuthMethodAny
}

func GetDomains(cfg *config.Config, remoteAddr, username, password string) map[string]struct{} {
	domainsAllowedDomains := getDomainsFromAllowedDomains(cfg.Auth.AllowedDomains, remoteAddr)
	if cfg.Auth.Method == config.AuthMethodAllowedDomains {
		return stripWildcards(domainsAllowedDomains)
	}

	domainsUsers := getDomainsFromUsers(cfg.Auth.Users, username, password)
	if cfg.Auth.Method == config.AuthMethodUsers {
		return stripWildcards(domainsUsers)
	}

	domains := map[string]struct{}{}
	switch cfg.Auth.Method {
	case config.AuthMethodBoth:
		for domain := range domainsAllowedDomains {
			if _, ok := domainsUsers[domain]; ok {
				domains[domain] = struct{}{}
				continue
			}
			for domainUser := range domainsUsers {
				if IsSubDomain(domainUser, domain) {
					domains[domainUser] = struct{}{}
				}
			}
		}
	case config.AuthMethodAny:
		maps.Copy(domains, domainsAllowedDomains)
		maps.Copy(domains, domainsUsers)
	}

	return stripWildcards(domains)
}

func getDomainsFromAllowedDomains(allowedDomains config.AllowedDomains, remoteAddr string) map[string]struct{} {
	domains := map[string]struct{}{}
	for domain, ipNets := range allowedDomains {
		for _, ipNet := range ipNets {
			ip := net.ParseIP(remoteAddr)
			if ip != nil && ipNet.Contains(ip) {
				domains[domain] = struct{}{}
				break
			}
		}
	}

	return domains
}

func getDomainsFromUsers(users []config.User, username, password string) map[string]struct{} {
	domains := map[string]struct{}{}
	if username == "" || password == "" {
		return domains
	}
	for _, user := range users {
		if constantTimeEqual(user.Username, username)&constantTimeEqual(user.Password, password) == 1 {
			for _, domain := range user.Domains {
				domains[domain] = struct{}{}
			}
		}
	}

	return domains
}

func stripWildcards(domains map[string]struct{}) map[string]struct{} {
	domainsStripped := map[string]struct{}{}
	for domain := range domains {
		domainsStripped[strings.TrimPrefix(domain, "*.")] = struct{}{}
	}

	return domainsStripped
}
