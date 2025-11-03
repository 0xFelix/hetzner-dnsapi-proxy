package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
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
		log.Printf("HEADER %+v", r.Header)
		next.ServeHTTP(w, r)
	})
}
