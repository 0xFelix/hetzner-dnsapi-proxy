package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("SetClientIP", func() {
	var (
		captured string
		inner    http.Handler
	)

	BeforeEach(func() {
		captured = ""
		inner = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			captured = r.RemoteAddr
		})
	})

	run := func(trustedProxies []netip.Prefix, remoteAddr string, headers map[string]string) *httptest.ResponseRecorder {
		handler := middleware.NewSetClientIP(trustedProxies)(inner)
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = remoteAddr
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	It("strips the port from a valid address", func() {
		rec := run(nil, "10.0.0.1:1234", nil)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("10.0.0.1"))
	})

	It("returns 500 on an unparseable remote address", func() {
		rec := run(nil, "not-an-address", nil)
		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
	})

	It("honors X-Real-Ip from a trusted proxy matched by bare IP", func() {
		rec := run(
			[]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")},
			"10.0.0.1:1234", map[string]string{"X-Real-Ip": "192.0.2.7"},
		)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("192.0.2.7"))
	})

	It("honors X-Real-Ip from a trusted proxy matched by CIDR", func() {
		rec := run(
			[]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
			"10.1.2.3:1234", map[string]string{"X-Real-Ip": "192.0.2.7"},
		)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("192.0.2.7"))
	})

	It("honors the first X-Forwarded-For entry from a trusted proxy when X-Real-Ip is absent", func() {
		rec := run(
			[]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")},
			"10.0.0.1:1234", map[string]string{"X-Forwarded-For": "192.0.2.7, 10.0.0.1"},
		)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("192.0.2.7"))
	})

	It("ignores forwarded headers from an untrusted proxy", func() {
		rec := run(nil, "10.0.0.1:1234", map[string]string{"X-Real-Ip": "192.0.2.7"})
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("10.0.0.1"))
	})

	It("ignores forwarded headers from an IP outside the trusted CIDR", func() {
		rec := run(
			[]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
			"11.0.0.1:1234", map[string]string{"X-Real-Ip": "192.0.2.7"},
		)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("11.0.0.1"))
	})

	DescribeTable("falls back to the proxy address on an invalid forwarded value",
		func(header, value string) {
			rec := run(
				[]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")},
				"10.0.0.1:1234", map[string]string{header: value},
			)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(captured).To(Equal("10.0.0.1"))
		},
		Entry("invalid X-Real-Ip", "X-Real-Ip", "not-an-ip"),
		Entry("X-Real-Ip with port", "X-Real-Ip", "192.0.2.7:80"),
		Entry("invalid first X-Forwarded-For", "X-Forwarded-For", "bogus, 192.0.2.7"),
		Entry("empty first X-Forwarded-For", "X-Forwarded-For", ", 192.0.2.7"),
	)

	It("normalizes IPv6 addresses", func() {
		rec := run(nil, "[2001:db8::1]:1234", nil)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(captured).To(Equal("2001:db8::1"))
	})
})
