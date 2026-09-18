package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("SplitFQDN", func() {
	DescribeTable(
		"should split successfully", func(fullName, expectedName, expectedZone string) {
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

var _ = Describe("JSON binding", func() {
	run := func(bind func(http.Handler) http.Handler, body string) (code int, called bool) {
		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			called = true
		})
		rec := httptest.NewRecorder()
		bind(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		return rec.Code, called
	}

	DescribeTable(
		"should accept valid bodies", func(bind func(http.Handler) http.Handler, body string) {
			code, called := run(bind, body)
			Expect(code).To(Equal(http.StatusOK))
			Expect(called).To(BeTrue())
		},
		Entry("httpreq", middleware.BindHTTPReq, `{"fqdn":"a.example.com","value":"x"}`),
		Entry("httpreq with unknown key", middleware.BindHTTPReq, `{"fqdn":"a.example.com","value":"x","other":1}`),
		Entry("acmedns", middleware.BindAcmeDNS, `{"subdomain":"a.example.com","txt":"x"}`),
	)

	DescribeTable(
		"should reject malformed bodies", func(bind func(http.Handler) http.Handler, body string) {
			code, called := run(bind, body)
			Expect(code).To(Equal(http.StatusBadRequest))
			Expect(called).To(BeFalse())
		},
		Entry("httpreq with duplicate key", middleware.BindHTTPReq, `{"fqdn":"a.example.com","fqdn":"b.example.com","value":"x"}`),
		Entry("httpreq with trailing data", middleware.BindHTTPReq, `{"fqdn":"a.example.com","value":"x"}{}`),
		Entry("httpreq with wrong-case key", middleware.BindHTTPReq, `{"FQDN":"a.example.com","value":"x"}`),
		Entry("httpreq with invalid UTF-8", middleware.BindHTTPReq, "{\"fqdn\":\"a.example.com\",\"value\":\"\xff\"}"),
		Entry("acmedns with duplicate key", middleware.BindAcmeDNS, `{"subdomain":"a.example.com","subdomain":"b.example.com","txt":"x"}`),
		Entry("acmedns with trailing data", middleware.BindAcmeDNS, `{"subdomain":"a.example.com","txt":"x"}{}`),
		Entry("acmedns with wrong-case key", middleware.BindAcmeDNS, `{"Subdomain":"a.example.com","txt":"x"}`),
		Entry("acmedns with invalid UTF-8", middleware.BindAcmeDNS, "{\"subdomain\":\"a.example.com\",\"txt\":\"\xff\"}"),
	)
})
