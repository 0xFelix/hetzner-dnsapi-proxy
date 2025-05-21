package middleware_test

import (
	"net/http"
	"net/http/httptest"

	// Original package under test
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContentTypeJSON Middleware", func() {
	var (
		rr                     *httptest.ResponseRecorder
		req                    *http.Request
		handler                http.Handler
		dummyNextHandlerCalled bool
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		dummyNextHandlerCalled = false
		dummyNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dummyNextHandlerCalled = true
			w.WriteHeader(http.StatusOK)
		})
		// Access middleware.ContentTypeJSON from the imported original package
		handler = middleware.ContentTypeJSON(dummyNextHandler)
	})

	Context("when Content-Type is valid (application/json)", func() {
		It("should call the next handler and return StatusOK", func() {
			req = httptest.NewRequest("POST", "/test", nil) // Method is irrelevant
			req.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(dummyNextHandlerCalled).To(BeTrue())
			Expect(rr.Body.String()).To(BeEmpty()) // Dummy handler writes no body
		})
	})

	Context("when Content-Type is invalid or missing", func() {
		type invalidCase struct {
			description       string
			contentTypeHeader string
			headerShouldBeSet bool // false if we want to test missing header
		}

		invalidCases := []invalidCase{
			{description: "application/json with charset", contentTypeHeader: "application/json; charset=utf-8", headerShouldBeSet: true},
			{description: "text/plain", contentTypeHeader: "text/plain", headerShouldBeSet: true},
			{description: "application/xml", contentTypeHeader: "application/xml", headerShouldBeSet: true},
			{description: "missing Content-Type header", contentTypeHeader: "", headerShouldBeSet: false},
		}

		for _, tc := range invalidCases {
			// Capture tc for the closure
			currentCase := tc
			It("should return StatusBadRequest for "+currentCase.description, func() {
				req = httptest.NewRequest("POST", "/test", nil)
				if currentCase.headerShouldBeSet {
					req.Header.Set("Content-Type", currentCase.contentTypeHeader)
				}
				// If !headerShouldBeSet, the header is omitted, testing the "missing header" case.

				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
				Expect(dummyNextHandlerCalled).To(BeFalse())
				Expect(rr.Body.String()).To(Equal("Content-Type must be application/json\n"))
			})
		}
	})
})
