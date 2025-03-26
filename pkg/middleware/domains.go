package middleware

import (
	"log"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func NewShowDomainsDirectAdmin(cfg *config.Config) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.AuthMethodIsValid(cfg.Auth.Method) {
				log.Printf("invalid auth method: %s", cfg.Auth.Method)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			username, password, _ := r.BasicAuth()
			values := url.Values{}
			for domain := range GetDomains(cfg, r.RemoteAddr, username, password) {
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

func GetDomains(cfg *config.Config, remoteAddr, username, password string) map[string]struct{} {
	domainsAllowedDomains := getDomainsFromAllowedDomains(cfg.Auth.AllowedDomains, remoteAddr)
	if cfg.Auth.Method == config.AuthMethodAllowedDomains {
		return domainsAllowedDomains
	}

	domainsUsers := getDomainsFromUsers(cfg.Auth.Users, username, password)
	if cfg.Auth.Method == config.AuthMethodUsers {
		return domainsUsers
	}

	domains := map[string]struct{}{}
	switch cfg.Auth.Method {
	case config.AuthMethodBoth:
		for domain := range domainsAllowedDomains {
			if _, ok := domainsUsers[domain]; ok {
				domains[domain] = struct{}{}
			}
		}
	case config.AuthMethodAny:
		maps.Copy(domains, domainsAllowedDomains)
		maps.Copy(domains, domainsUsers)
	}

	return domains
}

func getDomainsFromAllowedDomains(allowedDomains config.AllowedDomains, remoteAddr string) map[string]struct{} {
	domains := map[string]struct{}{}
	for domain, ipNets := range allowedDomains {
		for _, ipNet := range ipNets {
			ip := net.ParseIP(remoteAddr)
			if ip != nil && ipNet.Contains(ip) {
				domains[strings.TrimPrefix(domain, "*.")] = struct{}{}
				break
			}
		}
	}

	return domains
}

func getDomainsFromUsers(users []config.User, username, password string) map[string]struct{} {
	domains := map[string]struct{}{}
	for _, user := range users {
		if user.Username == username && user.Password == password {
			for _, domain := range user.Domains {
				domains[strings.TrimPrefix(domain, "*.")] = struct{}{}
			}
			break
		}
	}

	return domains
}
