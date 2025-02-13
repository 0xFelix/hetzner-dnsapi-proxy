package middleware

import (
	"log"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

func NewSetClientIP(trustedProxies []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addrPort, err := netip.ParseAddrPort(r.RemoteAddr)
			if err != nil {
				log.Printf("failed to parse remote address %s: %v", r.RemoteAddr, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.RemoteAddr = addrPort.Addr().String()
			if slices.Contains(trustedProxies, r.RemoteAddr) {
				ip := r.Header.Get("X-Real-Ip")
				if ip == "" {
					ipList := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
					ip = strings.TrimSpace(ipList[0])
				}
				if ip != "" {
					r.RemoteAddr = ip
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
