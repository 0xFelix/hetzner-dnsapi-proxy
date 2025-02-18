package middleware_test

import (
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("SplitFQDN", func() {
	DescribeTable("should split successfully a", func(fullName, expectedName, expectedZone string) {
		name, zone, err := middleware.SplitFQDN("test.example.com")
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
		name, zone, err := middleware.SplitFQDN("tld")
		Expect(err).To(MatchError("invalid fqdn: tld"))
		Expect(name).To(BeEmpty())
		Expect(zone).To(BeEmpty())
	})
})

var _ = Describe("DecodeBasicAuth", func() {
	It("should decode successfully", func() {
		const username = "username"
		const password = "password"
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		dUsername, dPassword, err := middleware.DecodeBasicAuth("Basic " + encoded)
		Expect(err).ToNot(HaveOccurred())
		Expect(dUsername).To(Equal(username))
		Expect(dPassword).To(Equal(password))
	})

	Context("should fail on", func() {
		It("an invalid authorization header", func() {
			username, password, err := middleware.DecodeBasicAuth("somethingsomething")
			Expect(err).To(MatchError(ContainSubstring("invalid authorization header: ")))
			Expect(username).To(BeEmpty())
			Expect(password).To(BeEmpty())
		})

		It("an invalid authorization method", func() {
			username, password, err := middleware.DecodeBasicAuth("NotBasic somethingsomething")
			Expect(err).To(MatchError(ContainSubstring("invalid authorization method: ")))
			Expect(username).To(BeEmpty())
			Expect(password).To(BeEmpty())
		})

		It("an invalid base64 value", func() {
			username, password, err := middleware.DecodeBasicAuth("Basic notbase64")
			Expect(err).To(MatchError(ContainSubstring("invalid base64 value: ")))
			Expect(username).To(BeEmpty())
			Expect(password).To(BeEmpty())
		})

		It("an invalid username and password", func() {
			encoded := base64.StdEncoding.EncodeToString([]byte("nocolon"))
			username, password, err := middleware.DecodeBasicAuth("Basic " + encoded)
			Expect(err).To(MatchError(ContainSubstring("invalid username and password: ")))
			Expect(username).To(BeEmpty())
			Expect(password).To(BeEmpty())
		})
	})
})
