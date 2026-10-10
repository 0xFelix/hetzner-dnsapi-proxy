package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
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

var _ = Describe("NormalizeFQDN", func() {
	DescribeTable(
		"should normalize", func(fqdn, expected string) {
			Expect(middleware.NormalizeFQDN(fqdn)).To(Equal(expected))
		},
		Entry("already normalized", "sub.example.com", "sub.example.com"),
		Entry("trailing dot", "sub.example.com.", "sub.example.com"),
		Entry("multiple trailing dots", "sub.example.com..", "sub.example.com"),
		Entry("upper case", "SUB.Example.COM", "sub.example.com"),
		Entry("upper case with trailing dot", "SUB.Example.COM.", "sub.example.com"),
	)
})

var _ = Describe("Binding of names", func() {
	bind := func(b func(http.Handler) http.Handler, r *http.Request) (*data.ReqData, int) {
		var reqData *data.ReqData
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			reqData, _ = data.ReqDataFromContext(r.Context())
		})
		rec := httptest.NewRecorder()
		b(next).ServeHTTP(rec, r)
		return reqData, rec.Code
	}

	newJSONRequest := func(body string) *http.Request {
		return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	}

	DescribeTable(
		"should bind acmedns requests", func(subdomain, expectedFullName, expectedName string) {
			reqData, code := bind(middleware.BindAcmeDNS, newJSONRequest(
				`{"subdomain":"`+subdomain+`","txt":"value"}`,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal(expectedFullName))
			Expect(reqData.Name).To(Equal(expectedName))
			Expect(reqData.Zone).To(Equal("example.com"))
		},
		Entry(
			"subdomain without prefix", "sub.example.com",
			"_acme-challenge.sub.example.com", "_acme-challenge.sub",
		),
		Entry(
			"subdomain with prefix", "_acme-challenge.sub.example.com",
			"_acme-challenge.sub.example.com", "_acme-challenge.sub",
		),
		Entry(
			"zone apex", "example.com",
			"_acme-challenge.example.com", "_acme-challenge",
		),
		Entry(
			"zone apex with trailing dot", "example.com.",
			"_acme-challenge.example.com", "_acme-challenge",
		),
		Entry(
			"upper case subdomain", "SUB.Example.COM",
			"_acme-challenge.sub.example.com", "_acme-challenge.sub",
		),
	)

	DescribeTable(
		"should normalize the name of plain requests", func(hostname string) {
			reqData, code := bind(middleware.BindPlain, httptest.NewRequest(
				http.MethodGet, "/?hostname="+hostname+"&ip=127.0.0.1", http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal("sub.example.com"))
			Expect(reqData.Name).To(Equal("sub"))
			Expect(reqData.Zone).To(Equal("example.com"))
		},
		Entry("trailing dot", "sub.example.com."),
		Entry("upper case", "SUB.Example.COM"),
	)

	DescribeTable(
		"should normalize the name of nic requests", func(hostname string) {
			reqData, code := bind(middleware.BindNicUpdate, httptest.NewRequest(
				http.MethodGet, "/?hostname="+hostname+"&myip=127.0.0.1", http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal("sub.example.com"))
			Expect(reqData.Name).To(Equal("sub"))
			Expect(reqData.Zone).To(Equal("example.com"))
		},
		Entry("trailing dot", "sub.example.com."),
		Entry("upper case", "SUB.Example.COM"),
	)

	DescribeTable(
		"should normalize the name of httpreq requests", func(fqdn string) {
			reqData, code := bind(middleware.BindHTTPReq, newJSONRequest(
				`{"fqdn":"`+fqdn+`","value":"value"}`,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal("sub.example.com"))
			Expect(reqData.Name).To(Equal("sub"))
		},
		Entry("trailing dot", "sub.example.com."),
		Entry("upper case", "SUB.Example.COM"),
	)

	DescribeTable(
		"should normalize the name of directadmin requests", func(domain, name string) {
			reqData, code := bind(middleware.BindDirectAdmin, httptest.NewRequest(
				http.MethodGet, "/?action=add&domain="+domain+"&name="+name+"&type=A&value=127.0.0.1", http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal("sub.example.com"))
			Expect(reqData.Name).To(Equal("sub"))
			Expect(reqData.Zone).To(Equal("example.com"))
		},
		Entry("trailing dot", "example.com.", "sub"),
		Entry("upper case", "Example.COM", "SUB"),
	)
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
