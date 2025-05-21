package middleware_test

import (
	"net/http"
	"net/http/httptest"

	// Original package under test
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetClientIP Middleware", func() {
	var (
		dummyHandler       http.HandlerFunc
		capturedRemoteAddr string
		rr                 *httptest.ResponseRecorder
		req                *http.Request
	)

	BeforeEach(func() {
		capturedRemoteAddr = "" // Reset for each test
		dummyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRemoteAddr = r.RemoteAddr
			w.WriteHeader(http.StatusOK)
		})
		rr = httptest.NewRecorder()
	})

	Context("when no proxies are trusted", func() {
		var (
			trustedProxies []string // nil
			mw             func(http.Handler) http.Handler
		)
		BeforeEach(func() {
			trustedProxies = nil
			mw = middleware.NewSetClientIP(trustedProxies)
		})

		It("should use r.RemoteAddr directly when no headers are present", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345"

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("10.0.0.1"))
		})

		It("should ignore X-Real-Ip and X-Forwarded-For headers", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			req.Header.Set("X-Real-Ip", "192.168.1.1")
			req.Header.Set("X-Forwarded-For", "172.16.0.1, 172.16.0.2")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("10.0.0.1"))
		})
	})

	Context("when proxies are trusted", func() {
		var (
			trustedProxies []string
			mw             func(http.Handler) http.Handler
		)
		BeforeEach(func() {
			trustedProxies = []string{"10.0.0.1"} // The proxy's IP
			mw = middleware.NewSetClientIP(trustedProxies)
		})

		It("should use X-Real-Ip if present and request is from a trusted proxy", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy
			req.Header.Set("X-Real-Ip", "192.168.1.1")
			req.Header.Set("X-Forwarded-For", "172.16.0.1") // Should be ignored

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("192.168.1.1"))
		})

		It("should use the first IP from X-Forwarded-For if X-Real-Ip is absent and request is from trusted proxy", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy
			req.Header.Set("X-Forwarded-For", "172.16.0.1, 10.0.0.5")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("172.16.0.1"))
		})
		
		It("should correctly parse the first IP from X-Forwarded-For even with spaces", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy
			req.Header.Set("X-Forwarded-For", "  172.16.0.1  , 172.16.0.2, 172.16.0.3")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("172.16.0.1"))
		})


		It("should use r.RemoteAddr (proxy's IP) if no headers are present, even if from trusted proxy", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("10.0.0.1"))
		})

		It("should ignore headers if request is not from a trusted proxy", func() {
			// trustedProxies is ["10.0.0.1"]
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.2:12345" // NOT from trusted proxy
			req.Header.Set("X-Real-Ip", "192.168.1.1")
			req.Header.Set("X-Forwarded-For", "172.16.0.1")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("10.0.0.2"))
		})
		
		It("should use a malformed IP from X-Forwarded-For if X-Real-Ip is absent", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy
			req.Header.Set("X-Forwarded-For", "malformed-ip,172.16.0.2")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("malformed-ip"))
		})

		It("should use proxy's IP if X-Forwarded-For contains only spaces", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345" // From trusted proxy
			req.Header.Set("X-Forwarded-For", "   ")

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(capturedRemoteAddr).To(Equal("10.0.0.1")) // Because "" from header is ignored
		})

	})

	Context("with invalid r.RemoteAddr format", func() {
		var (
			mw func(http.Handler) http.Handler
		)
		BeforeEach(func() {
			// No trusted proxies needed for this context, as parsing r.RemoteAddr happens first.
			mw = middleware.NewSetClientIP(nil)
		})

		It("should return HTTP 500 if r.RemoteAddr has no port", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1" // Malformed - no port

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(capturedRemoteAddr).To(BeEmpty()) // Handler should not be called
		})

		It("should return HTTP 500 if r.RemoteAddr is badly formatted", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "this:is:not:an:ip:port" // Malformed

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(capturedRemoteAddr).To(BeEmpty())
		})
		
		It("should return HTTP 500 if r.RemoteAddr contains only port", func() {
			req = httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ":12345" // Malformed

			handlerToTest := mw(dummyHandler)
			handlerToTest.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(capturedRemoteAddr).To(BeEmpty())
		})
	})
})
