package app_test

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	// Original package under test
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/app" // Import the original package
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"

	. "github.com/onsi/ginkgo/v2" // Ginkgo DSL
	. "github.com/onsi/gomega"    // Gomega matchers
)

var _ = Describe("App Logic", func() {

	Describe("LoggingResponseWriter", func() {
		It("should capture the status code correctly", func() {
			recorder := httptest.NewRecorder()
			// Assuming app.LoggingResponseWriter is now exported, and its StatusCode field is also exported.
			// And a constructor NewLoggingResponseWriter is available and exported.
			lrw := app.NewLoggingResponseWriter(recorder) // Use the new constructor

			expectedStatusCode := http.StatusAccepted
			lrw.WriteHeader(expectedStatusCode) // This will set lrw.StatusCode

			Expect(lrw.StatusCode).To(Equal(expectedStatusCode), "LoggingResponseWriter should have captured the status code.")
			Expect(recorder.Code).To(Equal(expectedStatusCode), "Underlying ResponseRecorder should also have the status code.")
		})
	})

	Describe("LogRequest Function", func() {
		var (
			logBuf         bytes.Buffer
			originalOutput io.Writer
			req            *http.Request
			start          time.Time
			statusCode     int
		)

		BeforeEach(func() {
			originalOutput = log.Writer() // Capture current log writer
			log.SetOutput(&logBuf)      // Redirect log output to buffer

			// Setup common test data
			req = httptest.NewRequest("GET", "/test-url", nil)
			req.RemoteAddr = "1.2.3.4:12345"
			start = time.Now().Add(-1 * time.Second) // 1 second ago
			statusCode = http.StatusOK
		})

		It("should format log messages as expected", func() {
			// Assuming logRequest is made exportable as app.LogRequest
			app.LogRequest(req, start, statusCode)

			logOutput := logBuf.String()
			actualLogMessage := ""
			// log.Printf adds a timestamp prefix, e.g., "2023/10/27 10:00:00 "
			// We need to strip this prefix before matching.
			if parts := strings.SplitN(logOutput, " ", 3); len(parts) == 3 {
				actualLogMessage = parts[2]
			} else {
				// Fail the test if the log output isn't what we expect (missing prefix)
				Fail("Log output was not in the expected format (missing date/time prefix): " + logOutput)
			}

			// Regex from the original test
			expectedLogPattern := `^\| 200 \| {5,11}[0-9]\.[0-9]{1,7}s \| {2,10}1\.2\.3\.4:12345 \| GET {5}"/test-url"\n$`
			Expect(actualLogMessage).To(MatchRegexp(expectedLogPattern))
		})

		AfterEach(func() {
			log.SetOutput(originalOutput) // Restore original log writer
			logBuf.Reset()                // Clear buffer for next test
		})
	})

	Describe("RouteHandling", func() {
		var (
			cfg        *config.Config
			appHandler http.Handler // Store the app handler
			rr         *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			// Setup config
			cfg = &config.Config{
				Token: "test-token",
				Auth: config.Auth{
					Method:         config.AuthMethodAny,
					AllowedDomains: make(config.AllowedDomains),
					Users:          []config.User{{Username: "user", Password: "password", Domains: []string{"example.com"}}},
				},
				TrustedProxies: []string{"192.0.2.1"},
				Debug:          false,
			}
			_, ipNetAll, err := net.ParseCIDR("0.0.0.0/0")
			Expect(err).NotTo(HaveOccurred(), "Failed to parse CIDR") // Use Gomega for error checking
			cfg.Auth.AllowedDomains["example.com"] = []*net.IPNet{ipNetAll}

			// Create app handler instance
			appHandler = app.New(cfg) // app.New is the constructor from the original package
			rr = httptest.NewRecorder() // Initialize response recorder for each test
		})

		// Define test cases struct (can be local to Describe or outside if shared)
		type routeTestCase struct {
			name       string
			method     string
			path       string
			body       string
			username   string
			password   string
			remoteAddr string
			headers    map[string]string
			wantStatus int
		}

		// Test cases data
		testCases := []routeTestCase{
			{name: "GET /plain/update - success", method: http.MethodGet, path: "/plain/update?zone_name=example.com&record_type=A&value=1.2.3.4", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "POST /acmedns/update - success", method: http.MethodPost, path: "/acmedns/update", body: `{"subdomain":"test.example.com","txt":"abc"}`, headers: map[string]string{"Content-Type": "application/json"}, remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "POST /httpreq/present - success", method: http.MethodPost, path: "/httpreq/present", body: `{"fqdn":"test.example.com","value":"dns-value"}`, headers: map[string]string{"Content-Type": "application/json"}, remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "POST /httpreq/cleanup - success", method: http.MethodPost, path: "/httpreq/cleanup", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "GET /directadmin/CMD_API_SHOW_DOMAINS - success", method: http.MethodGet, path: "/directadmin/CMD_API_SHOW_DOMAINS", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "GET /directadmin/CMD_API_DOMAIN_POINTER - success", method: http.MethodGet, path: "/directadmin/CMD_API_DOMAIN_POINTER", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "GET /directadmin/CMD_API_DNS_CONTROL - success", method: http.MethodGet, path: "/directadmin/CMD_API_DNS_CONTROL?domain=example.com&action=add&type=A&name=test&value=1.2.3.4", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusOK},
			{name: "GET /unknown/route - not found", method: http.MethodGet, path: "/unknown/route", remoteAddr: "192.0.2.1:12345", wantStatus: http.StatusNotFound},
			{name: "GET /plain/update - forbidden domain", method: http.MethodGet, path: "/plain/update?zone_name=otherdomain.com&record_type=A&value=1.2.3.4", remoteAddr: "1.2.3.4:54321", wantStatus: http.StatusUnauthorized},
		}

		// Iterate over test cases and create an It block for each
		for _, tc := range testCases {
			// Capture the test case for the closure, to avoid tc being the same in all It blocks
			currentTest := tc
			It(currentTest.name, func() {
				var reqBody io.Reader
				if currentTest.body != "" {
					reqBody = strings.NewReader(currentTest.body)
				}

				req := httptest.NewRequest(currentTest.method, currentTest.path, reqBody)

				if currentTest.remoteAddr != "" {
					req.RemoteAddr = currentTest.remoteAddr
				}
				if currentTest.username != "" && currentTest.password != "" {
					req.SetBasicAuth(currentTest.username, currentTest.password)
				}
				if currentTest.headers != nil {
					for k, v := range currentTest.headers {
						req.Header.Set(k, v)
					}
				}

				appHandler.ServeHTTP(rr, req) // Use the appHandler from BeforeEach

				// Use Gomega matcher for status code
				// Provide rr.Body.String() as additional context for failures
				Expect(rr.Code).To(Equal(currentTest.wantStatus), "Response body: "+rr.Body.String())
			})
		}
	})
})
