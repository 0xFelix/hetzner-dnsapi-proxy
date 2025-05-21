package middleware_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	// Original package under test
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogDebug Middleware", func() {
	var (
		originalLogOutput    io.Writer
		logBuffer            *bytes.Buffer
		rr                   *httptest.ResponseRecorder
		req                  *http.Request
		handler              http.Handler
		dummyNextHandler     http.HandlerFunc
		nextHandlerCalled    bool
		bodyReadByNext       []byte
		testBody             string
		testHeaderKey        string
		testHeaderValue      string
		anotherTestHeaderKey string
		anotherTestHeaderValue string
	)

	BeforeEach(func() {
		// Setup log capture
		originalLogOutput = log.Writer()
		logBuffer = &bytes.Buffer{}
		log.SetOutput(logBuffer)

		// Initialize common vars
		rr = httptest.NewRecorder()
		nextHandlerCalled = false
		bodyReadByNext = nil
		testBody = "Hello, this is a test body!"
		testHeaderKey = "X-Test-Header"
		testHeaderValue = "TestValue123"
		anotherTestHeaderKey = "Content-Type" // Using a common header for variety
		anotherTestHeaderValue = "text/plain"


		dummyNextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
			var err error
			bodyReadByNext, err = io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred(), "Next handler should be able to read the body")
			w.WriteHeader(http.StatusOK)
		})

		// Access middleware.LogDebug from the imported original package
		handler = middleware.LogDebug(dummyNextHandler)
	})

	AfterEach(func() {
		log.SetOutput(originalLogOutput)
	})

	Context("with a non-empty request body and headers", func() {
		BeforeEach(func() {
			req = httptest.NewRequest("POST", "/debug", strings.NewReader(testBody))
			req.Header.Set(testHeaderKey, testHeaderValue)
			req.Header.Set(anotherTestHeaderKey, anotherTestHeaderValue)
			handler.ServeHTTP(rr, req)
		})

		It("should call the next handler", func() {
			Expect(nextHandlerCalled).To(BeTrue())
		})

		It("should allow the next handler to successfully complete (status OK)", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("should allow the next handler to re-read the request body", func() {
			Expect(string(bodyReadByNext)).To(Equal(testBody))
		})

		It("should log the request body", func() {
			Expect(logBuffer.String()).To(ContainSubstring("BODY " + testBody))
		})

		It("should log the request headers, including custom and standard ones", func() {
			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("HEADER map["))
			Expect(logOutput).To(ContainSubstring(testHeaderKey + ":[" + testHeaderValue + "]"))
			Expect(logOutput).To(ContainSubstring(anotherTestHeaderKey + ":[" + anotherTestHeaderValue + "]"))
		})
	})

	Context("with an empty request body", func() {
		BeforeEach(func() {
			testBody = "" // Ensure testBody is empty for this context's assertions
			req = httptest.NewRequest("POST", "/debug-empty", strings.NewReader(testBody))
			req.Header.Set(testHeaderKey, "emptyTest") // A header for the empty body request
			handler.ServeHTTP(rr, req)
		})

		It("should call the next handler", func() {
			Expect(nextHandlerCalled).To(BeTrue())
		})

		It("should allow the next handler to successfully complete (status OK)", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("should allow the next handler to re-read the empty request body", func() {
			Expect(string(bodyReadByNext)).To(Equal(""))
		})

		It("should log an empty request body", func() {
			// The log format is "BODY <content>", so for empty it's "BODY "
			Expect(logBuffer.String()).To(ContainSubstring("BODY "))
            // To be more precise, ensure it doesn't log "BODY \n" if the body was truly empty vs. just a newline.
            // The current middleware logs `string(bodyBytes)` which for an empty body is `""`.
            // So, "BODY " followed by the log timestamp or next log line.
            // A regex might be better if we want to be super strict about what follows "BODY ".
            // For now, `ContainSubstring("BODY ")` and not `ContainSubstring("BODY \n")` (unless newline is part of actual empty body log)
		})

		It("should log the request headers for an empty body request", func() {
			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("HEADER map["))
			Expect(logOutput).To(ContainSubstring(testHeaderKey + ":[emptyTest]"))
		})
	})
	
	// Note: Testing the "failed to read request body" case in the middleware
    // would require providing a faulty io.Reader as r.Body, which is more involved
    // and typically done using custom mock types for r.Body.
    // This was not part of the original test's scope and is not included here.
})
