package middleware_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/middleware"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/model"
)

var _ = Describe("CheckPermission", func() {
	DescribeTable("should allow access", func(cfg *config.Config, data *model.ReqData, remoteAddr string) {
		Expect(middleware.CheckPermission(cfg, data, remoteAddr)).To(BeTrue())
	},
		Entry("with auth method allowedDomains",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodAllowedDomains,
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
			},
			"127.0.0.1",
		),
		Entry("with auth method users",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodUsers,
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"",
		),
		Entry("with auth method both",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodBoth,
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with auth method any and allowed domains",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodAny,
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
			},
			"127.0.0.1",
		),
		Entry("with auth method any and users",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodAny,
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"",
		),
	)

	DescribeTable("should deny access", func(cfg *config.Config, data *model.ReqData, remoteAddr string) {
		Expect(middleware.CheckPermission(cfg, data, remoteAddr)).To(BeFalse())
	},
		Entry("with auth method allowedDomains and missing allowed domains",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodAllowedDomains,
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with auth method users and missing users",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodUsers,
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"",
		),
		Entry("with auth method both and missing allowed domains",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodBoth,
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with auth method both and missing users",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodBoth,
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with auth method both and missing allowed domains and users",
			&config.Config{
				Auth: config.Auth{
					Method: config.AuthMethodBoth,
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with empty auth method",
			&config.Config{
				Auth: config.Auth{
					Method: "",
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
		Entry("with invalid auth method",
			&config.Config{
				Auth: config.Auth{
					Method: "invalid",
					AllowedDomains: config.AllowedDomains{"example.com": []*net.IPNet{
						{
							IP:   net.IPv4(127, 0, 0, 1),
							Mask: net.IPv4Mask(255, 255, 255, 255),
						},
					}},
					Users: []config.User{{
						Username: "username",
						Password: "password",
						Domains:  []string{"example.com"},
					}},
				},
			},
			&model.ReqData{
				FullName: "example.com",
				Username: "username",
				Password: "password",
			},
			"127.0.0.1",
		),
	)
})

var _ = Describe("CheckAllowedDomains", func() {
	DescribeTable("should allow access", func(fqdn, clientIP string, allowedDomains config.AllowedDomains) {
		Expect(middleware.CheckAllowedDomains(fqdn, clientIP, allowedDomains)).To(BeTrue())
	},
		Entry("with wildcard and matching host", "example.com", "127.0.0.1",
			config.AllowedDomains{"*": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("with wildcard and matching ipnet", "example.com", "192.168.0.1",
			config.AllowedDomains{"*": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
		Entry("when domain equals fqdn and matching host", "example.com", "127.0.0.1",
			config.AllowedDomains{"example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when domain equals fqdn and matching ipnet", "example.com", "192.168.0.1",
			config.AllowedDomains{"example.com": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
		Entry("when domain is a subdomain and matching host", "sub.example.com", "127.0.0.1",
			config.AllowedDomains{"*.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when domain is a subdomain and matching ipnet", "sub.example.com", "192.168.0.1",
			config.AllowedDomains{"*.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
	)

	DescribeTable("should deny access", func(fqdn, clientIP string, allowedDomains config.AllowedDomains) {
		Expect(middleware.CheckAllowedDomains(fqdn, clientIP, allowedDomains)).To(BeFalse())
	},
		Entry("with wildcard and non matching host", "example.com", "127.0.0.2",
			config.AllowedDomains{"*": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("with wildcard and non matching ipnet", "example.com", "127.0.0.1",
			config.AllowedDomains{"*": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
		Entry("with wildcard and invalid client ip", "example.com", "127.0.0.x",
			config.AllowedDomains{"*": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when domain does not match and matching host", "test.com", "127.0.0.1",
			config.AllowedDomains{"example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when domain does not match and matching ipnet", "test.com", "192.168.0.1",
			config.AllowedDomains{"example.com": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
		Entry("with matching domain and invalid client ip", "example.com", "127.0.0.x",
			config.AllowedDomains{"example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when subdomain does not match and matching host", "sub.test.com", "127.0.0.1",
			config.AllowedDomains{"*.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when subdomain does not match and matching ipnet", "sub.test.com", "192.168.0.1",
			config.AllowedDomains{"*.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(192, 168, 0, 0),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
		),
		Entry("with matching subdomain and invalid client ip", "sub.example.com", "127.0.0.x",
			config.AllowedDomains{"sub.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
		Entry("when subdomains do not match", "test.example.com", "127.0.0.1",
			config.AllowedDomains{"sub.example.com": []*net.IPNet{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}},
		),
	)
})

var _ = Describe("CheckUsers", func() {
	DescribeTable("should allow access", func(fqdn, username, password string, users []config.User) {
		Expect(middleware.CheckUsers(fqdn, username, password, users)).To(BeTrue())
	},
		Entry("with matching credentials and wildcard", "example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("with matching credentials and fqdn equals domain", "example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"example.com"},
			}},
		),
		Entry("with matching credentials and fqdn equals one of the domains", "example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"test.com", "example.com"},
			}},
		),
		Entry("with matching credentials and fqdn is a subdomain", "sub.example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*.example.com"},
			}},
		),
		Entry("with matching credentials and fqdn is a subdomain of one of the domains", "sub.example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"test.com", "*.example.com"},
			}},
		),
	)

	DescribeTable("should deny access", func(fqdn, username, password string, users []config.User) {
		Expect(middleware.CheckUsers(fqdn, username, password, users)).To(BeFalse())
	},
		Entry("when username does not match", "example.com", "something", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("when password does not match", "example.com", "username", "something",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("when domain does not match", "test.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"example.com"},
			}},
		),
		Entry("when subdomain does not match", "sub.test.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*.example.com"},
			}},
		),
		Entry("when subdomains do not match", "test.example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"sub.example.com"},
			}},
		),
		Entry("when fqdn is empty", "", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("when username is empty", "example.com", "", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("when password is empty", "example.com", "username", "",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  []string{"*"},
			}},
		),
		Entry("when domains are empty", "example.com", "username", "password",
			[]config.User{{
				Username: "username",
				Password: "password",
				Domains:  nil,
			}},
		),
	)
})

var _ = Describe("IsSubDomain", func() {
	DescribeTable("should return true", func(sub, parent string) {
		Expect(middleware.IsSubDomain(sub, parent)).To(BeTrue())
	},
		Entry("when sub is a subdomain", "sub.example.com", "*.example.com"),
		Entry("when sub is a double subdomain", "subsub.sub.example.com", "*.example.com"),
	)

	DescribeTable("should return false", func(sub, parent string) {
		Expect(middleware.IsSubDomain(sub, parent)).To(BeFalse())
	},
		Entry("when parent does not begin with wildcard", "sub.example.com", "example.com"),
		Entry("when sub has fewer parts than parent", "sub.example.com", "*.sub.example.com"),
		Entry("when sub does not match parent", "sub.test.com", "*.example.com"),
	)
})
