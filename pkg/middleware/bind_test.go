package middleware_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("SplitFQDN", func() {
	DescribeTable("should split successfully", func(fullName, expectedName, expectedZone string) {
		name, zone, err := middleware.SplitFQDN(fullName)
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal(expectedName))
		Expect(zone).To(Equal(expectedZone))
	},
		Entry("simple domain", "example.com", "", "example.com"),
		Entry("single subdomain", "test.example.com", "test", "example.com"),
		Entry("double subdomain", "sub.test.example.com", "sub.test", "example.com"),
		Entry("triple subdomain", "subsub.sub.test.example.com", "subsub.sub.test", "example.com"),
		Entry("multi-part TLD (co.uk)", "example.co.uk", "", "example.co.uk"),
		Entry("subdomain on multi-part TLD (co.uk)", "test.example.co.uk", "test", "example.co.uk"),
		Entry("multi-part TLD (com.au)", "example.com.au", "", "example.com.au"),
		Entry("subdomain on multi-part TLD (com.au)", "test.example.com.au", "test", "example.com.au"),
	)

	It("should fail on TLD", func() {
		name, zone, err := middleware.SplitFQDN("tld")
		Expect(err).To(MatchError("invalid fqdn: tld"))
		Expect(name).To(BeEmpty())
		Expect(zone).To(BeEmpty())
	})
})
