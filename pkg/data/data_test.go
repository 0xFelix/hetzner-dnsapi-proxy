package data_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
)

var _ = Describe("SplitFQDN", func() {
	DescribeTable("should split successfully a", func(fullName, expectedName, expectedZone string) {
		name, zone, err := data.SplitFQDN("test.example.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal("test"))
		Expect(zone).To(Equal("example.com"))
	},
		Entry("simple domain", "example.com", "", "example.com"),
		Entry("single subdomain", "test.example.com", "test", "example.com"),
		Entry("double subdomain", "sub.test.example.com", "sub.test", "example.com"),
		Entry("triple subdomain", "subsub.sub.test.example.com", "subsub.sub.test", "example.com"),
	)

	It("should fail on TLD", func() {
		name, zone, err := data.SplitFQDN("tld")
		Expect(err).To(MatchError("invalid fqdn: tld"))
		Expect(name).To(BeEmpty())
		Expect(zone).To(BeEmpty())
	})
})
