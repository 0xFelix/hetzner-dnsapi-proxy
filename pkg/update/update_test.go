package update_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/update"
)

var _ = Describe("CheckPermissions", func() {
	DescribeTable("should allow access", func(fqdn, clientIP string, allowedDomains func() config.AllowedDomains) {
		Expect(update.CheckPermissions(fqdn, clientIP, allowedDomains())).To(BeTrue())
	},
		Entry("with wildcard and matching host", "example.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("with wildcard and matching ipnet", "example.com", "192.168.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain equals fqdn and matching host", "example.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain equals fqdn and matching ipnet", "example.com", "192.168.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain is a subdomain and matching host", "sub.example.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*.example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain is a subdomain and matching ipnet", "sub.example.com", "192.168.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*.example.com": []*net.IPNet{ipNet}}
			},
		),
	)

	DescribeTable("should deny access", func(fqdn, clientIP string, allowedDomains func() config.AllowedDomains) {
		Expect(update.CheckPermissions(fqdn, clientIP, allowedDomains())).To(BeFalse())
	},
		Entry("with wildcard and non matching host", "example.com", "127.0.0.2",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("with wildcard and non matching ipnet", "example.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("with wildcard and invalid client ip", "example.com", "127.0.0.x",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain does not match and matching host", "test.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when domain does not match and matching ipnet", "test.com", "192.168.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("with matching domain and invalid client ip", "example.com", "127.0.0.x",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when subdomain does not match and matching host", "sub.test.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*.example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when subdomain does not match and matching ipnet", "sub.test.com", "192.168.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*.example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("with matching subdomain and invalid client ip", "sub.example.com", "127.0.0.x",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"sub.example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("when subdomains do not match", "test.example.com", "127.0.0.1",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"sub.example.com": []*net.IPNet{ipNet}}
			},
		),
	)
})

var _ = Describe("IsSubDomain", func() {
	DescribeTable("should return true", func(sub, parent string) {
		Expect(update.IsSubDomain(sub, parent)).To(BeTrue())
	},
		Entry("when sub is a subdomain", "sub.example.com", "*.example.com"),
		Entry("when sub is a double subdomain", "subsub.sub.example.com", "*.example.com"),
	)

	DescribeTable("should return false", func(sub, parent string) {
		Expect(update.IsSubDomain(sub, parent)).To(BeFalse())
	},
		Entry("when parent does not begin with wildcard", "sub.example.com", "example.com"),
		Entry("when sub has fewer parts than parent", "sub.example.com", "*.sub.example.com"),
		Entry("when sub does not match parent", "sub.test.com", "*.example.com"),
	)
})
