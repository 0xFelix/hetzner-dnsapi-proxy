package middleware_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/middleware"
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
