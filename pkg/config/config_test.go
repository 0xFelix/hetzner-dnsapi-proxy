package config_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

var _ = Describe("AllowedDomains", func() {
	const unexpectedPartsCountErr = "failed to parse allowed domain, length of parts != 2"

	DescribeTable("should unmarshal successfully", func(text string, expected func() config.AllowedDomains) {
		allowedDomains := config.AllowedDomains{}
		Expect(allowedDomains.UnmarshalText([]byte(text))).To(Succeed())
		Expect(allowedDomains).To(Equal(expected()))
	},
		Entry("wildcard for localhost", "*,127.0.0.1/32",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("wildcard for remote host", "*,192.168.0.0/16",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("domain for host", "example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("two entries", "*,127.0.0.1/32;example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"*":           []*net.IPNet{ipNetLocalhost},
					"example.com": []*net.IPNet{ipNetRemote},
				}
			},
		),
		Entry("three entries", "*,127.0.0.1/32;example.com,192.168.0.1/32;test.com,127.0.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"*":           []*net.IPNet{ipNetLocalhost},
					"example.com": []*net.IPNet{ipNetRemote},
					"test.com":    []*net.IPNet{ipNetLocalhost},
				}
			},
		),
		Entry("multiple entries for same domain", "example.com,127.0.0.1/32;example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"example.com": []*net.IPNet{ipNetLocalhost, ipNetRemote},
				}
			},
		),
	)

	DescribeTable("should fail", func(text, expected string) {
		allowedDomains := config.AllowedDomains{}
		Expect(allowedDomains.UnmarshalText([]byte(text))).To(MatchError(expected))
		Expect(allowedDomains).To(BeEmpty())
	},
		Entry("on empty", "", unexpectedPartsCountErr),
		Entry("on empty after entry", "*,127.0.0.1/32;", unexpectedPartsCountErr),
		Entry("on empty before entry", ";*,127.0.0.1/32", unexpectedPartsCountErr),
		Entry("on empty between entries", "*,127.0.0.1/32;;*,127.0.0.1/32", unexpectedPartsCountErr),
		Entry("on invalid CIDR", "*,127.0.0.1/64;", "invalid CIDR address: 127.0.0.1/64"),
	)
})
