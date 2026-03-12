package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

func LogDebug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		body, err := io.ReadAll(io.TeeReader(r.Body, &buf))
		if err != nil {
			log.Printf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(&buf)
		log.Printf("BODY %s", string(body))
		header := sanitize.LogValue(fmt.Sprintf("%+v", r.Header))
		//nolint:gosec // value is sanitized above
		log.Printf("HEADER %s", header)
		next.ServeHTTP(w, r)
	})
}
