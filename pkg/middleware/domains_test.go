package middleware_test

import (
	"encoding/base64"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("GetDomains", func() {
	const (
		remoteAddr = "127.0.0.1"
		username   = "username"
		password   = "password"
	)

	var authorization string

	BeforeEach(func() {
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	})

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
				},
				Users: []config.User{
					{
						Username: username,
						Password: password,
						Domains:  []string{"something.com", "nice.com"},
					},
					{
						Username: "someone",
						Password: "somepassword",
						Domains:  []string{"greatwebsite.com"},
					},
				},
			},
		}
		domains, err := middleware.GetDomains(cfg, remoteAddr, authorization)
		Expect(err).ToNot(HaveOccurred())
		Expect(domains).To(Equal(expectedDomains))
	},
		Entry("with auth method allowed domains",
			config.AuthMethodAllowedDomains,
			map[string]struct{}{
				"example.com": {},
				"nice.com":    {},
			},
		),
		Entry("with auth method users",
			config.AuthMethodUsers,
			map[string]struct{}{
				"something.com": {},
				"nice.com":      {},
			},
		),
		Entry("with auth method both",
			config.AuthMethodBoth,
			map[string]struct{}{
				"nice.com": {},
			},
		),
		Entry("with auth method any",
			config.AuthMethodAny,
			map[string]struct{}{
				"example.com":   {},
				"nice.com":      {},
				"something.com": {},
			},
		),
	)

	It("should fail if auth method is invalid", func() {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: "invalid",
			},
		}
		domains, err := middleware.GetDomains(cfg, remoteAddr, authorization)
		Expect(err).To(MatchError("invalid auth method: invalid"))
		Expect(domains).To(BeEmpty())
	})

	DescribeTable("should not fail if basic auth is invalid and auth method is", func(authMethod string) {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: authMethod,
			},
		}
		domains, err := middleware.GetDomains(cfg, remoteAddr, "something")
		Expect(err).ToNot(HaveOccurred())
		Expect(domains).To(BeEmpty())
	},
		Entry("allowedDomains", config.AuthMethodAllowedDomains),
		Entry("any", config.AuthMethodAny),
	)

	DescribeTable("should fail if basic auth is invalid and auth method is", func(authMethod string) {
		cfg := &config.Config{
			Auth: config.Auth{
				Method: authMethod,
			},
		}
		domains, err := middleware.GetDomains(cfg, remoteAddr, "something")
		Expect(err).To(MatchError("invalid authorization header: something"))
		Expect(domains).To(BeEmpty())
	},
		Entry("users", config.AuthMethodUsers),
		Entry("both", config.AuthMethodBoth),
	)
})
