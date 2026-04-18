package middleware_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
)

var _ = Describe("LogDebug", func() {
	var (
		logBuf      *bytes.Buffer
		prevOutput  io.Writer
		prevFlags   int
		innerCalled bool
		inner       http.Handler
	)

	BeforeEach(func() {
		logBuf = &bytes.Buffer{}
		prevOutput = log.Writer()
		prevFlags = log.Flags()
		log.SetOutput(logBuf)
		log.SetFlags(0)

		innerCalled = false
		inner = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			innerCalled = true
			_, _ = io.ReadAll(r.Body)
		})
	})

	AfterEach(func() {
		log.SetOutput(prevOutput)
		log.SetFlags(prevFlags)
	})

	It("redacts sensitive headers", func() {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Basic c2VjcmV0")
		req.Header.Set("X-Api-User", "admin")
		req.Header.Set("X-Api-Key", "supersecret")
		req.Header.Set("User-Agent", "probe/1.0")
		rec := httptest.NewRecorder()
		middleware.LogDebug(inner).ServeHTTP(rec, req)

		Expect(innerCalled).To(BeTrue())
		Expect(rec.Code).To(Equal(http.StatusOK))

		logged := logBuf.String()
		Expect(logged).NotTo(ContainSubstring("c2VjcmV0"))
		Expect(logged).NotTo(ContainSubstring("supersecret"))
		Expect(logged).NotTo(ContainSubstring("admin"))
		Expect(logged).To(ContainSubstring("[REDACTED]"))
		Expect(logged).To(ContainSubstring("probe/1.0"))
	})

	It("returns 413 on a body that exceeds the limit", func() {
		body := strings.Repeat("A", 2<<10) // 2 KB, over the 1 KB limit
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		rec := httptest.NewRecorder()
		middleware.LogDebug(inner).ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusRequestEntityTooLarge))
		Expect(innerCalled).To(BeFalse())
	})

	It("passes the body through to the next handler", func() {
		const payload = "hello"
		var received []byte
		capture := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			Expect(err).ToNot(HaveOccurred())
			received = b
		})

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		middleware.LogDebug(capture).ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(string(received)).To(Equal(payload))
	})
})
