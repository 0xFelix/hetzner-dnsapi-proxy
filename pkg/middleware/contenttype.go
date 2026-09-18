package middleware

import (
	"fmt"
	"mime"
	"net/http"
)

const (
	headerContentType     = "Content-Type"
	applicationJSON       = "application/json"
	applicationURLEncoded = "application/x-www-form-urlencoded"
)

func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mediaType, _, err := mime.ParseMediaType(r.Header.Get(headerContentType)); err != nil || mediaType != applicationJSON {
			http.Error(w, fmt.Sprintf("%s must be %s", headerContentType, applicationJSON), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
