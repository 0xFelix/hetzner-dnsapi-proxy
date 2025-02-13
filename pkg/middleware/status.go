package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

const failedWriteResponseFmt = "failed to write response: %v"

func StatusOk(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func StatusOkAcmeDNS(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := reqDataFromContext(r.Context())
		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resData, err := json.Marshal(map[string]string{
			"txt": data.Value,
		})
		if err != nil {
			log.Printf("failed to marshal response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(headerContentType, applicationJSON)
		if _, err := w.Write(resData); err != nil {
			log.Printf(failedWriteResponseFmt, err)
		}
	})
}

func StatusOkDirectAdmin(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := url.Values{
			"error": []string{"0"},
			"text":  []string{"OK"},
		}

		w.Header().Set(headerContentType, applicationURLEncoded)
		if _, err := w.Write([]byte(values.Encode())); err != nil {
			log.Printf(failedWriteResponseFmt, err)
		}
	})
}
