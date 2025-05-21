package middleware // Keep original package to access unexported symbols

import (
	"context"
	"encoding/json" // Still needed for constructing expected JSON for MatchJSON with struct
	"net/http"
	"net/http/httptest"
	// "net/url" // Not strictly needed if only comparing final string for DirectAdmin
	"strings" // Still useful for TrimSpace

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// Note: "github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware" is not needed for self-package tests.
)

var _ = Describe("Status Handlers", func() {
	var (
		rr                *httptest.ResponseRecorder
		req               *http.Request
		nextHandlerCalled bool
		dummyNextHandler  http.HandlerFunc
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/test", nil) // Basic request, customize per test if needed
		nextHandlerCalled = false
		dummyNextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
		})
	})

	Describe("StatusOk Handler", func() {
		var handler http.Handler
		BeforeEach(func() {
			handler = StatusOk(dummyNextHandler)
			handler.ServeHTTP(rr, req)
		})

		It("should return HTTP StatusOK", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
		})
		It("should return an empty body", func() {
			Expect(rr.Body.String()).To(BeEmpty())
		})
		It("should not call the next handler", func() {
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

	Describe("StatusOkAcmeDNS Handler", func() {
		var handler http.Handler
		BeforeEach(func() {
			handler = StatusOkAcmeDNS(dummyNextHandler)
		})

		Context("when ReqData is in context", func() {
			var (
				reqData      *ReqData
				expectedJSON string
			)
			BeforeEach(func() {
				reqData = &ReqData{
					FullName: "test.example.com",
					Value:    "dns-challenge-value",
				}
				ctx := context.Background()
				ctxWithReqData := newContextWithReqData(ctx, reqData) // Uses unexported newContextWithReqData
				req = req.WithContext(ctxWithReqData)

				// Prepare expected JSON string based on actual implementation {"txt": data.Value}
				expectedMap := map[string]string{"txt": reqData.Value}
				jsonBytes, err := json.Marshal(expectedMap)
				Expect(err).NotTo(HaveOccurred())
				expectedJSON = string(jsonBytes)

				handler.ServeHTTP(rr, req)
			})

			It("should return HTTP StatusOK", func() {
				Expect(rr.Code).To(Equal(http.StatusOK))
			})
			It("should set Content-Type to application/json", func() {
				Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))
			})
			It("should return the correct JSON body", func() {
				Expect(rr.Body.String()).To(MatchJSON(expectedJSON))
			})
			It("should not call the next handler", func() {
				Expect(nextHandlerCalled).To(BeFalse())
			})
		})

		Context("when ReqData is not in context", func() {
			BeforeEach(func() {
				// req uses default context.Background() from outer BeforeEach
				handler.ServeHTTP(rr, req)
			})

			It("should return HTTP StatusInternalServerError", func() {
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			})
			It("should not call the next handler", func() {
				Expect(nextHandlerCalled).To(BeFalse())
			})
		})
	})

	Describe("StatusOkDirectAdmin Handler", func() {
		var handler http.Handler
		BeforeEach(func() {
			handler = StatusOkDirectAdmin(dummyNextHandler)
			handler.ServeHTTP(rr, req)
		})

		It("should return HTTP StatusOK", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
		})
		It("should set Content-Type to application/x-www-form-urlencoded", func() {
			Expect(rr.Header().Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
		})
		It("should return the correct form-urlencoded body", func() {
			// url.Values.Encode() sorts keys, so "error" will come before "text".
			Expect(strings.TrimSpace(rr.Body.String())).To(Equal("error=0&text=OK"))
		})
		It("should not call the next handler", func() {
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})
})
