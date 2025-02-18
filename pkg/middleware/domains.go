package middleware

import (
	"fmt"
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
			domains, err := GetDomains(cfg, r.RemoteAddr, r.Header.Get(headerAuthorization))
			if err != nil {
				log.Printf("%v", err)
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

func GetDomains(cfg *config.Config, remoteAddr, authorization string) (map[string]struct{}, error) {
	if !config.AuthMethodIsValid(cfg.Auth.Method) {
		return nil, fmt.Errorf("invalid auth method: %s", cfg.Auth.Method)
	}

	domainsAllowedDomains := getDomainsFromAllowedDomains(cfg.Auth.AllowedDomains, remoteAddr)
	if cfg.Auth.Method == config.AuthMethodAllowedDomains {
		return domainsAllowedDomains, nil
	}

	domainsUsers, err := getDomainsFromUsers(cfg.Auth.Users, authorization)
	if err != nil &&
		(cfg.Auth.Method == config.AuthMethodUsers || cfg.Auth.Method == config.AuthMethodBoth) {
		return nil, err
	}
	if cfg.Auth.Method == config.AuthMethodUsers {
		return domainsUsers, nil
	}

	domains := map[string]struct{}{}
	if cfg.Auth.Method == config.AuthMethodBoth {
		for domain := range domainsAllowedDomains {
			if _, ok := domainsUsers[domain]; ok {
				domains[domain] = struct{}{}
			}
		}
	} else if cfg.Auth.Method == config.AuthMethodAny {
		maps.Copy(domains, domainsAllowedDomains)
		maps.Copy(domains, domainsUsers)
	}

	return domains, nil
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

func getDomainsFromUsers(users []config.User, authorization string) (map[string]struct{}, error) {
	username, password, err := DecodeBasicAuth(authorization)
	if err != nil {
		return nil, err
	}

	domains := map[string]struct{}{}
	for _, user := range users {
		if user.Username == username && user.Password == password {
			for _, domain := range user.Domains {
				domains[strings.TrimPrefix(domain, "*.")] = struct{}{}
			}
			break
		}
	}

	return domains, nil
}
