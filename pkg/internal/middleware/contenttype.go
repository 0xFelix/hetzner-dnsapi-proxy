package middleware

import (
	"fmt"
	"net/http"
)

const (
	headerContentType     = "Content-Type"
	applicationJSON       = "application/json"
	applicationURLEncoded = "application/x-www-form-urlencoded"
)

func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerContentType) != applicationJSON {
			http.Error(w, fmt.Sprintf("%s must be %s", headerContentType, applicationJSON), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
