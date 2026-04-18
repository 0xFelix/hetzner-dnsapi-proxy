package middleware_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("SecurityHeaders", func() {
	It("sets security headers on the response", func() {
		handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
		Expect(rec.Header().Get("X-Frame-Options")).To(Equal("DENY"))
		Expect(rec.Header().Get("Content-Security-Policy")).To(Equal("default-src 'none'"))
		Expect(rec.Header().Get("Cache-Control")).To(Equal("no-store"))
	})
})
