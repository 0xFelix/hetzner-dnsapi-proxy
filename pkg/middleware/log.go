package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

var redactedHeaders = []string{"Authorization", "X-Api-User", "X-Api-Key"}

func LogDebug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		body, err := io.ReadAll(io.TeeReader(r.Body, &buf))
		if err != nil {
			log.Printf("failed to read request body: %v", err)
			status := http.StatusInternalServerError
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				status = http.StatusRequestEntityTooLarge
			}
			w.WriteHeader(status)
			return
		}
		r.Body = io.NopCloser(&buf)
		log.Printf("BODY %s", string(body))
		header := sanitize.LogValue(fmt.Sprintf("%+v", redactHeader(r.Header)))
		//nolint:gosec // value is sanitized above
		log.Printf("HEADER %s", header)
		next.ServeHTTP(w, r)
	})
}

func redactHeader(h http.Header) http.Header {
	clone := h.Clone()
	for _, key := range redactedHeaders {
		if len(clone.Values(key)) > 0 {
			clone.Set(key, "[REDACTED]")
		}
	}
	return clone
}
