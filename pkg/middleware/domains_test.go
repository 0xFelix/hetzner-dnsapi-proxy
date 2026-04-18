package middleware_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/ratelimit"
)

var _ = Describe("GetDomains", func() {
	const (
		remoteAddr = "127.0.0.1"
		username   = "username"
		password   = "password"
	)

	DescribeTable("should successfully return expected domains", func(authMethod string, expectedDomains map[string]struct{}) {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: authMethod,
				AllowedDomains: config.AllowedDomains{
					"example.com": []*net.IPNet{{
						IP:   net.IPv4(127, 0, 0, 1),
						Mask: net.IPv4Mask(255, 255, 255, 255),
					}},
					"test.com": []*net.IPNet{{
						IP:   net.IPv4(192, 168, 0, 1),
						Mask: net.IPv4Mask(255, 255, 0, 0),
					}},
					"nice.com": []*net.IPNet{{
						IP:   net.IPv4(127, 0, 0, 1),
						Mask: net.IPv4Mask(255, 255, 255, 255),
					}},
					"*.parent.com": []*net.IPNet{{
						IP:   net.IPv4(127, 0, 0, 1),
						Mask: net.IPv4Mask(255, 255, 255, 255),
					}},
				},
				Users: []config.User{
					{
						Username: username,
						Password: password,
						Domains:  []string{"something.com", "nice.com", "sub.parent.com"},
					},
					{
						Username: "someone",
						Password: "somepassword",
						Domains:  []string{"greatwebsite.com"},
					},
				},
			},
		}
		Expect(middleware.GetDomains(cfg, remoteAddr, username, password)).To(Equal(expectedDomains))
	},
		Entry("with auth method allowed domains",
			config.AuthMethodAllowedDomains,
			map[string]struct{}{
				"example.com": {},
				"nice.com":    {},
				"parent.com":  {},
			},
		),
		Entry("with auth method users",
			config.AuthMethodUsers,
			map[string]struct{}{
				"something.com":  {},
				"nice.com":       {},
				"sub.parent.com": {},
			},
		),
		Entry("with auth method both",
			config.AuthMethodBoth,
			map[string]struct{}{
				"nice.com":       {},
				"sub.parent.com": {},
			},
		),
		Entry("with auth method any",
			config.AuthMethodAny,
			map[string]struct{}{
				"example.com":    {},
				"nice.com":       {},
				"something.com":  {},
				"parent.com":     {},
				"sub.parent.com": {},
			},
		),
	)

	It("should return nothing if auth method is invalid", func() {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: "invalid",
			},
		}
		Expect(middleware.GetDomains(cfg, remoteAddr, username, password)).To(BeEmpty())
	})

	DescribeTable("should return something if basic auth is invalid and auth method is", func(authMethod string) {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: authMethod,
			},
		}
		Expect(middleware.GetDomains(cfg, remoteAddr, "", "")).To(BeEmpty())
	},
		Entry("allowedDomains", config.AuthMethodAllowedDomains),
		Entry("any", config.AuthMethodAny),
	)

	DescribeTable("should return nothing if basic auth is invalid and auth method is", func(authMethod string) {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: authMethod,
			},
		}
		Expect(middleware.GetDomains(cfg, remoteAddr, "", "")).To(BeEmpty())
	},
		Entry("users", config.AuthMethodUsers),
		Entry("both", config.AuthMethodBoth),
	)
})

var _ = Describe("NewShowDomainsDirectAdmin", func() {
	const (
		ip       = "127.0.0.1"
		username = "username"
		password = "password"
	)

	var (
		lockout *ratelimit.Lockout
		cfg     *config.Config
	)

	BeforeEach(func() {
		lockout = ratelimit.NewLockout(3, time.Hour, 15*time.Minute)
		cfg = &config.Config{
			Auth: config.Auth{
				Method: config.AuthMethodUsers,
				Users: []config.User{{
					Username: username,
					Password: password,
					Domains:  []string{"example.com"},
				}},
			},
		}
	})

	run := func(username, password string) *httptest.ResponseRecorder {
		handler := middleware.NewShowDomainsDirectAdmin(cfg, lockout)(nil)
		req := httptest.NewRequest(http.MethodGet, "/directadmin/CMD_API_SHOW_DOMAINS", http.NoBody)
		req.RemoteAddr = ip
		if username != "" || password != "" {
			req.SetBasicAuth(username, password)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	It("returns 429 when the client is locked out", func() {
		for range 3 {
			lockout.RecordFailure(ip)
		}
		Expect(lockout.IsBlocked(ip)).To(BeTrue())

		rec := run(username, password)
		Expect(rec.Code).To(Equal(http.StatusTooManyRequests))
	})

	It("returns 200 with the domain list when authenticated", func() {
		rec := run(username, password)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(Equal("list=example.com"))
	})

	It("returns 401 with WWW-Authenticate on bad credentials in users mode", func() {
		rec := run(username, "wrong")
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		Expect(rec.Header().Get("WWW-Authenticate")).To(Equal(`Basic realm="Restricted"`))
		Expect(rec.Body.String()).To(BeEmpty())
	})

	It("returns 401 without WWW-Authenticate in allowedDomains mode on IP mismatch", func() {
		cfg.Auth.Method = config.AuthMethodAllowedDomains
		cfg.Auth.AllowedDomains = config.AllowedDomains{
			"example.com": []*net.IPNet{{
				IP:   net.IPv4(10, 0, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 255),
			}},
		}
		rec := run("", "")
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		Expect(rec.Header().Get("WWW-Authenticate")).To(BeEmpty())
	})

	It("records a failure on bad credentials when the auth method uses users", func() {
		run(username, "wrong")
		run(username, "wrong")
		Expect(lockout.IsBlocked(ip)).To(BeFalse())
		run(username, "wrong")
		Expect(lockout.IsBlocked(ip)).To(BeTrue())
	})

	It("resets the failure counter after a successful auth", func() {
		run(username, "wrong")
		run(username, "wrong")

		rec := run(username, password)
		Expect(rec.Code).To(Equal(http.StatusOK))

		// Two more bad attempts must still not trigger the lockout because
		// the counter was reset.
		run(username, "wrong")
		run(username, "wrong")
		Expect(lockout.IsBlocked(ip)).To(BeFalse())
	})

	It("does not record failures in allowedDomains mode", func() {
		cfg.Auth.Method = config.AuthMethodAllowedDomains
		cfg.Auth.AllowedDomains = config.AllowedDomains{}

		for range 5 {
			run(username, "wrong")
		}
		Expect(lockout.IsBlocked(ip)).To(BeFalse())
	})

	It("does not record a failure when no credentials are supplied", func() {
		cfg.Auth.Method = config.AuthMethodAny
		cfg.Auth.AllowedDomains = config.AllowedDomains{}

		for range 5 {
			run("", "")
		}
		Expect(lockout.IsBlocked(ip)).To(BeFalse())
	})

	It("returns 500 on an invalid auth method", func() {
		cfg.Auth.Method = "bogus"
		rec := run(username, password)
		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
	})
})
