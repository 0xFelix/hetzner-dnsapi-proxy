package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
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
		Entry("underscore labels", "_sip._tcp.example.com", "_sip._tcp", "example.com"),
		Entry("wildcard", "*.example.com", "*", "example.com"),
		Entry("wildcard subdomain", "*.sub.example.com", "*.sub", "example.com"),
		Entry("maximum length", strings.Repeat("a.", 121)+"example.com", strings.Repeat("a.", 120)+"a", "example.com"),
	)

	DescribeTable(
		"should reject invalid names", func(fqdn string) {
			name, zone, err := middleware.SplitFQDN(fqdn)
			Expect(err).To(HaveOccurred())
			Expect(name).To(BeEmpty())
			Expect(zone).To(BeEmpty())
		},
		Entry("empty", ""),
		Entry("slash", "a/b.example.com"),
		Entry("dot segment", "../x.example.com"),
		Entry("question mark", "a?b.example.com"),
		Entry("percent", "a%2fb.example.com"),
		Entry("space", "a b.example.com"),
		Entry("empty label", "a..example.com"),
		Entry("leading dot", ".example.com"),
		Entry("leading hyphen", "-a.example.com"),
		Entry("trailing hyphen", "a-.example.com"),
		Entry("too long label", strings.Repeat("a", 64)+".example.com"),
		Entry("too long name", strings.Repeat("a.", 122)+"example.com"),
		Entry("too long wildcard name", "*."+strings.Repeat("a.", 121)+"example.com"),
		Entry("wildcard not leftmost", "a.*.example.com"),
		Entry("partial wildcard label", "a*.example.com"),
		Entry("bare wildcard", "*"),
		Entry("wildcard zone", "*.com"),
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
	bindWithBody := func(b func(http.Handler) http.Handler, r *http.Request) (*data.ReqData, int, string) {
		var reqData *data.ReqData
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			reqData, _ = data.ReqDataFromContext(r.Context())
		})
		rec := httptest.NewRecorder()
		b(next).ServeHTTP(rec, r)
		return reqData, rec.Code, rec.Body.String()
	}

	bind := func(b func(http.Handler) http.Handler, r *http.Request) (*data.ReqData, int) {
		reqData, code, _ := bindWithBody(b, r)
		return reqData, code
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

	DescribeTable(
		"should reject invalid names", func(b func(http.Handler) http.Handler, r *http.Request, expectedCode int) {
			reqData, code, body := bindWithBody(b, r)
			Expect(code).To(Equal(expectedCode))
			Expect(reqData).To(BeNil())
			if expectedCode == http.StatusOK {
				Expect(body).To(Equal("notfqdn"))
			}
		},
		Entry("plain", middleware.BindPlain, httptest.NewRequest(
			http.MethodGet, "/?hostname=a%2Fb.example.com&ip=127.0.0.1", http.NoBody,
		), http.StatusBadRequest),
		Entry("nic", middleware.BindNicUpdate, httptest.NewRequest(
			http.MethodGet, "/?hostname=a%3Fb.example.com&myip=127.0.0.1", http.NoBody,
		), http.StatusOK),
		Entry("httpreq", middleware.BindHTTPReq, newJSONRequest(
			`{"fqdn":"../x.example.com","value":"v"}`,
		), http.StatusBadRequest),
		Entry("acmedns", middleware.BindAcmeDNS, newJSONRequest(
			`{"subdomain":"a%2fb.example.com","txt":"v"}`,
		), http.StatusBadRequest),
		Entry("acmedns with wildcard", middleware.BindAcmeDNS, newJSONRequest(
			`{"subdomain":"*.example.com","txt":"v"}`,
		), http.StatusBadRequest),
		Entry("acmedns too long with prefix", middleware.BindAcmeDNS, newJSONRequest(
			`{"subdomain":"`+strings.Repeat("a.", 121)+`example.com","txt":"v"}`,
		), http.StatusBadRequest),
		Entry("directadmin", middleware.BindDirectAdmin, httptest.NewRequest(
			http.MethodGet, "/?action=add&domain=example.com&name=a%2Fb&type=A&value=127.0.0.1", http.NoBody,
		), http.StatusBadRequest),
		Entry("records", middleware.BindRecords, newJSONRequest(
			`{"fqdn":"a b.example.com","type":"A","value":"127.0.0.1"}`,
		), http.StatusBadRequest),
	)

	DescribeTable(
		"should bind records requests", func(fqdn, recordType, value, expectedFullName, expectedName, expectedType, expectedValue string) {
			reqData, code := bind(middleware.BindRecords, newJSONRequest(
				`{"fqdn":"`+fqdn+`","type":"`+recordType+`","value":"`+value+`"}`,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.FullName).To(Equal(expectedFullName))
			Expect(reqData.Name).To(Equal(expectedName))
			Expect(reqData.Zone).To(Equal("example.com"))
			Expect(reqData.Type).To(Equal(expectedType))
			Expect(reqData.Value).To(Equal(expectedValue))
		},
		Entry("A", "sub.example.com", "A", "127.0.0.1", "sub.example.com", "sub", "A", "127.0.0.1"),
		Entry("A with ipv4-mapped ipv6 address", "sub.example.com", "A", "::ffff:127.0.0.1", "sub.example.com", "sub", "A", "127.0.0.1"),
		Entry("AAAA", "sub.example.com", "AAAA", "::1", "sub.example.com", "sub", "AAAA", "::1"),
		Entry("AAAA with non-canonical address", "sub.example.com", "AAAA", "0:0::1", "sub.example.com", "sub", "AAAA", "::1"),
		Entry("TXT at the zone apex", "example.com", "TXT", "some text", "example.com", "", "TXT", "some text"),
		Entry("wildcard", "*.example.com", "A", "127.0.0.1", "*.example.com", "*", "A", "127.0.0.1"),
		Entry(
			"lower case type", "sub.example.com", "cname", "target.example.com",
			"sub.example.com", "sub", "CNAME", "target.example.com.",
		),
		Entry(
			"CNAME", "sub.example.com", "CNAME", "target.example.com",
			"sub.example.com", "sub", "CNAME", "target.example.com.",
		),
		Entry(
			"CNAME with absolute upper case target", "SUB.example.com.", "CNAME", "Target.Example.COM.",
			"sub.example.com", "sub", "CNAME", "target.example.com.",
		),
		Entry(
			"CNAME with underscore", "sub.example.com", "CNAME", "_dkim.example.com",
			"sub.example.com", "sub", "CNAME", "_dkim.example.com.",
		),
		Entry(
			"SRV", "_sip._tcp.example.com", "SRV", "10 5 5060 sip.example.com",
			"_sip._tcp.example.com", "_sip._tcp", "SRV", "10 5 5060 sip.example.com.",
		),
		Entry(
			"SRV with extra whitespace", "_sip._tcp.example.com", "SRV", ` 10  5 5060\tSIP.example.com. `,
			"_sip._tcp.example.com", "_sip._tcp", "SRV", "10 5 5060 sip.example.com.",
		),
		Entry(
			"SRV with maximum values", "_sip._tcp.example.com", "SRV", "65535 65535 65535 sip.example.com",
			"_sip._tcp.example.com", "_sip._tcp", "SRV", "65535 65535 65535 sip.example.com.",
		),
		Entry(
			"SRV with leading zeros", "_sip._tcp.example.com", "SRV", "00 05 05060 sip.example.com",
			"_sip._tcp.example.com", "_sip._tcp", "SRV", "0 5 5060 sip.example.com.",
		),
		Entry(
			"SRV with unavailable service", "_sip._tcp.example.com", "SRV", "0 0 0 .",
			"_sip._tcp.example.com", "_sip._tcp", "SRV", "0 0 0 .",
		),
	)

	DescribeTable(
		"should select the mode of records requests", func(recordType, value, replace string, expected data.Mode) {
			reqData, code := bind(middleware.BindRecords, newJSONRequest(
				`{"fqdn":"_sip._tcp.example.com","type":"`+recordType+`","value":"`+value+`"`+replace+`}`,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.Mode).To(Equal(expected))
			Expect(reqData.CheckConflicts).To(BeTrue())
		},
		Entry("SRV appends", "SRV", "10 5 5060 sip.example.com", "", data.ModeAppend),
		Entry("SRV with replace false appends", "SRV", "10 5 5060 sip.example.com", `,"replace":false`, data.ModeAppend),
		Entry("SRV with replace replaces", "SRV", "10 5 5060 sip.example.com", `,"replace":true`, data.ModeReplace),
		Entry("SRV with unavailable service replaces", "SRV", "0 0 0 .", "", data.ModeReplace),
		Entry("SRV with unavailable service and replace false replaces", "SRV", "0 0 0 .", `,"replace":false`, data.ModeReplace),
		Entry("TXT replaces", "TXT", "text", "", data.ModeReplace),
		Entry("TXT with replace replaces", "TXT", "text", `,"replace":true`, data.ModeReplace),
		Entry("A replaces", "A", "127.0.0.1", "", data.ModeReplace),
		Entry("CNAME replaces", "CNAME", "target.example.com", "", data.ModeReplace),
	)

	DescribeTable(
		"should not check conflicts or append for other endpoints", func(b func(http.Handler) http.Handler, r *http.Request) {
			reqData, code := bind(b, r)
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.Mode).To(Equal(data.ModeReplace))
			Expect(reqData.CheckConflicts).To(BeFalse())
		},
		Entry("plain", middleware.BindPlain, httptest.NewRequest(http.MethodGet, "/?hostname=sub.example.com&ip=127.0.0.1", http.NoBody)),
		Entry("nic", middleware.BindNicUpdate, httptest.NewRequest(http.MethodGet, "/?hostname=sub.example.com&myip=127.0.0.1", http.NoBody)),
		Entry("acmedns", middleware.BindAcmeDNS, newJSONRequest(`{"subdomain":"sub.example.com","txt":"v"}`)),
		Entry("directadmin", middleware.BindDirectAdmin, httptest.NewRequest(
			http.MethodGet, "/?action=add&domain=example.com&name=sub&type=TXT&value=v", http.NoBody,
		)),
	)

	It("should append values of httpreq requests without checking conflicts", func() {
		reqData, code := bind(middleware.BindHTTPReq, newJSONRequest(`{"fqdn":"sub.example.com","value":"v"}`))
		Expect(code).To(Equal(http.StatusOK))
		Expect(reqData.Mode).To(Equal(data.ModeAppend))
		Expect(reqData.CheckConflicts).To(BeFalse())
	})

	DescribeTable(
		"should reject invalid records requests", func(body string) {
			reqData, code := bind(middleware.BindRecords, newJSONRequest(body))
			Expect(code).To(Equal(http.StatusBadRequest))
			Expect(reqData).To(BeNil())
		},
		Entry("missing fqdn", `{"type":"A","value":"127.0.0.1"}`),
		Entry("missing type", `{"fqdn":"sub.example.com","value":"127.0.0.1"}`),
		Entry("missing value", `{"fqdn":"sub.example.com","type":"A"}`),
		Entry("unsupported type", `{"fqdn":"sub.example.com","type":"MX","value":"10 mx.example.com"}`),
		Entry("invalid fqdn", `{"fqdn":"com","type":"A","value":"127.0.0.1"}`),
		Entry("AAAA with zone", `{"fqdn":"sub.example.com","type":"AAAA","value":"fe80::1%eth0"}`),
		Entry("A with ipv6 address", `{"fqdn":"sub.example.com","type":"A","value":"::1"}`),
		Entry("AAAA with ipv4 address", `{"fqdn":"sub.example.com","type":"AAAA","value":"127.0.0.1"}`),
		Entry("CNAME at the zone apex", `{"fqdn":"example.com","type":"CNAME","value":"target.example.com"}`),
		Entry("CNAME with space in target", `{"fqdn":"sub.example.com","type":"CNAME","value":"a b.example.com"}`),
		Entry("CNAME with empty label", `{"fqdn":"sub.example.com","type":"CNAME","value":"a..example.com"}`),
		Entry("CNAME with leading hyphen", `{"fqdn":"sub.example.com","type":"CNAME","value":"-a.example.com"}`),
		Entry("CNAME with too long label", `{"fqdn":"sub.example.com","type":"CNAME","value":"`+strings.Repeat("a", 64)+`.example.com"}`),
		Entry("CNAME with too long target", `{"fqdn":"sub.example.com","type":"CNAME","value":"`+strings.Repeat("a.", 127)+`com"}`),
		Entry("SRV with too few fields", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 sip.example.com"}`),
		Entry("SRV with too many fields", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 5060 sip.example.com x"}`),
		Entry("SRV with negative priority", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"-1 5 5060 sip.example.com"}`),
		Entry("SRV with non-numeric weight", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 x 5060 sip.example.com"}`),
		Entry("SRV with port out of range", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 65536 sip.example.com"}`),
		Entry("SRV with invalid target", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 5060 sip..example.com"}`),
		Entry("CNAME with multiple trailing dots", `{"fqdn":"sub.example.com","type":"CNAME","value":"a.example.com.."}`),
		Entry("SRV with multiple trailing dots", `{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 5060 sip.example.com.."}`),
	)

	DescribeTable(
		"should filter records requests by record type", func(path string, allowed []string, body string, expected int) {
			// The type check runs after the authorizer, which is left out here,
			// so every request counts as authorized.
			bindAndCheck := func(next http.Handler) http.Handler {
				return middleware.BindRecords(middleware.NewRecordTypeAllowed(&config.Config{RecordTypes: allowed})(next))
			}
			reqData, code := bind(bindAndCheck, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
			Expect(code).To(Equal(expected))
			Expect(reqData != nil).To(Equal(expected == http.StatusOK))
		},
		Entry(
			"allowed type", "/records/present", []string{config.RecordTypeCNAME, config.RecordTypeTXT},
			`{"fqdn":"sub.example.com","type":"cname","value":"target.example.com"}`, http.StatusOK,
		),
		Entry(
			"type not in list on present", "/records/present", []string{config.RecordTypeCNAME},
			`{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 5060 sip.example.com"}`, http.StatusForbidden,
		),
		Entry(
			"type not in list on cleanup", "/records/cleanup", []string{config.RecordTypeCNAME},
			`{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"10 5 5060 sip.example.com"}`, http.StatusForbidden,
		),
		Entry(
			"empty list", "/records/present", []string{},
			`{"fqdn":"sub.example.com","type":"A","value":"127.0.0.1"}`, http.StatusForbidden,
		),
		Entry(
			"malformed value is rejected by bind first", "/records/present", []string{config.RecordTypeCNAME},
			`{"fqdn":"_sip._tcp.example.com","type":"SRV","value":"garbage"}`, http.StatusBadRequest,
		),
		Entry(
			"unsupported type is rejected by bind first", "/records/present", []string{},
			`{"fqdn":"sub.example.com","type":"MX","value":"10 mx.example.com"}`, http.StatusBadRequest,
		),
	)

	It("should log denied record types without leaking the value", func() {
		logBuf := &bytes.Buffer{}
		prevOutput := log.Writer()
		prevFlags := log.Flags()
		log.SetOutput(logBuf)
		log.SetFlags(0)
		DeferCleanup(func() {
			log.SetOutput(prevOutput)
			log.SetFlags(prevFlags)
		})

		req := httptest.NewRequest(http.MethodPost, "/records/present", strings.NewReader(
			`{"fqdn":"sub.example.com","type":"CNAME","value":"secret-target.example.com"}`,
		))
		req.RemoteAddr = "192.0.2.1:1234"
		_, code := bind(func(next http.Handler) http.Handler {
			return middleware.BindRecords(middleware.NewRecordTypeAllowed(&config.Config{RecordTypes: []string{}})(next))
		}, req)
		Expect(code).To(Equal(http.StatusForbidden))
		Expect(logBuf.String()).To(Equal(
			"client '192.0.2.1:1234' is not allowed to set 'CNAME' records of 'sub.example.com'\n",
		))
	})

	It("should fail without request data in the context", func() {
		handler := middleware.NewRecordTypeAllowed(&config.Config{RecordTypes: config.SupportedRecordTypes()})(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
		)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newJSONRequest(`{}`))
		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
	})

	DescribeTable(
		"should normalize the ip of plain requests", func(ip, expectedIP, expectedType string) {
			reqData, code := bind(middleware.BindPlain, httptest.NewRequest(
				http.MethodGet, "/?hostname=sub.example.com&ip="+ip, http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.Value).To(Equal(expectedIP))
			Expect(reqData.Type).To(Equal(expectedType))
		},
		Entry("ipv4", "127.0.0.1", "127.0.0.1", "A"),
		Entry("ipv4-mapped ipv6", "::ffff:127.0.0.1", "127.0.0.1", "A"),
		Entry("non-canonical ipv6", "0:0::1", "::1", "AAAA"),
	)

	It("should reject plain ips with a zone", func() {
		reqData, code := bind(middleware.BindPlain, httptest.NewRequest(
			http.MethodGet, "/?hostname=sub.example.com&ip=fe80::1%25eth0", http.NoBody,
		))
		Expect(code).To(Equal(http.StatusBadRequest))
		Expect(reqData).To(BeNil())
	})

	DescribeTable(
		"should normalize the ip of nic requests", func(ip, expectedIP, expectedType string) {
			reqData, code := bind(middleware.BindNicUpdate, httptest.NewRequest(
				http.MethodGet, "/?hostname=sub.example.com&myip="+ip, http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.Value).To(Equal(expectedIP))
			Expect(reqData.Type).To(Equal(expectedType))
		},
		Entry("ipv4", "127.0.0.1", "127.0.0.1", "A"),
		Entry("ipv4-mapped ipv6", "::ffff:127.0.0.1", "127.0.0.1", "A"),
		Entry("non-canonical ipv6", "0:0::1", "::1", "AAAA"),
	)

	It("should reject nic ips with a zone", func() {
		reqData, code, body := bindWithBody(middleware.BindNicUpdate, httptest.NewRequest(
			http.MethodGet, "/?hostname=sub.example.com&myip=fe80::1%25eth0", http.NoBody,
		))
		Expect(code).To(Equal(http.StatusOK))
		Expect(body).To(ContainSubstring("notfqdn"))
		Expect(reqData).To(BeNil())
	})

	DescribeTable(
		"should normalize the value of directadmin requests", func(recordType, value, expectedValue string) {
			reqData, code := bind(middleware.BindDirectAdmin, httptest.NewRequest(
				http.MethodGet, "/?action=add&domain=example.com&name=sub&type="+recordType+"&value="+value, http.NoBody,
			))
			Expect(code).To(Equal(http.StatusOK))
			Expect(reqData.Value).To(Equal(expectedValue))
		},
		Entry("ipv4-mapped ipv6 as A", "A", "::ffff:127.0.0.1", "127.0.0.1"),
		Entry("non-canonical AAAA", "AAAA", "0:0::1", "::1"),
	)

	DescribeTable(
		"should reject directadmin ips with a zone", func(recordType, value string) {
			reqData, code := bind(middleware.BindDirectAdmin, httptest.NewRequest(
				http.MethodGet, "/?action=add&domain=example.com&name=sub&type="+recordType+"&value="+value, http.NoBody,
			))
			Expect(code).To(Equal(http.StatusBadRequest))
			Expect(reqData).To(BeNil())
		},
		Entry("ipv4-mapped ipv6 with zone as A", "A", "::ffff:127.0.0.1%25eth0"),
		Entry("AAAA with zone", "AAAA", "fe80::1%25eth0"),
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
		Entry("records", middleware.BindRecords, `{"fqdn":"a.example.com","type":"TXT","value":"x"}`),
		Entry("records with unknown key", middleware.BindRecords, `{"fqdn":"a.example.com","type":"TXT","value":"x","other":1}`),
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
		Entry(
			"records with duplicate key", middleware.BindRecords,
			`{"fqdn":"a.example.com","fqdn":"b.example.com","type":"TXT","value":"x"}`,
		),
		Entry("records with trailing data", middleware.BindRecords, `{"fqdn":"a.example.com","type":"TXT","value":"x"}{}`),
		Entry("records with wrong-case key", middleware.BindRecords, `{"FQDN":"a.example.com","type":"TXT","value":"x"}`),
		Entry("records with invalid UTF-8", middleware.BindRecords, "{\"fqdn\":\"a.example.com\",\"type\":\"TXT\",\"value\":\"\xff\"}"),
	)
})
