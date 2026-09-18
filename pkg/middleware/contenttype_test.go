package middleware_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("ContentTypeJSON", func() {
	DescribeTable(
		"should check the media type", func(contentType string, expected int) {
			handler := middleware.ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(expected))
		},
		Entry("plain", "application/json", http.StatusOK),
		Entry("with charset", "application/json; charset=utf-8", http.StatusOK),
		Entry("mixed case", "Application/JSON", http.StatusOK),
		Entry("empty", "", http.StatusBadRequest),
		Entry("other type", "text/plain", http.StatusBadRequest),
		Entry("malformed", "application/json/x", http.StatusBadRequest),
	)
})
