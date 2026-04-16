package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/ratelimit"
)

const (
	nicTokenGood    = "good"
	nicTokenNotFQDN = "notfqdn"
	nicTokenBadAuth = "badauth"
	nicTokenNoHost  = "nohost"
	nicTokenDNSErr  = "dnserr"
	nicTokenAbuse   = "abuse"
	nicToken911     = "911"
	textPlainUTF8   = "text/plain; charset=utf-8"
)

func BindNicUpdate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		if err := r.ParseForm(); err != nil {
			log.Printf(failedParseRequestFmt, err)
			writeNicToken(w, http.StatusOK, nicTokenNotFQDN)
			return
		}

		hostname := r.Form.Get("hostname")
		if hostname == "" {
			writeNicToken(w, http.StatusOK, nicTokenNotFQDN)
			return
		}

		ip := r.Form.Get("myip")
		if ip == "" {
			ip = r.RemoteAddr
		}

		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			writeNicToken(w, http.StatusOK, nicTokenNotFQDN)
			return
		}

		recordType := recordTypeA
		if parsedIP.To4() == nil {
			recordType = recordTypeAAAA
		}

		name, zone, err := SplitFQDN(hostname)
		if err != nil {
			writeNicToken(w, http.StatusOK, nicTokenNotFQDN)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(w, r.WithContext(
			data.NewContextWithReqData(r.Context(),
				&data.ReqData{
					FullName:  hostname,
					Name:      name,
					Zone:      zone,
					Value:     ip,
					Type:      recordType,
					Username:  username,
					Password:  password,
					BasicAuth: true,
				},
			)),
		)
	})
}

func NicAuth(cfg *config.Config, lockout *ratelimit.Lockout) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqData, err := data.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				writeNicToken(w, http.StatusOK, nicToken911)
				return
			}

			if lockout.IsBlocked(r.RemoteAddr) {
				logLockedOut(r.RemoteAddr)
				writeNicToken(w, http.StatusOK, nicTokenAbuse)
				return
			}

			if CheckPermission(cfg, reqData, r.RemoteAddr) {
				lockout.Reset(r.RemoteAddr)
				next.ServeHTTP(w, r)
				return
			}

			logPermissionDenied(r.RemoteAddr, reqData)
			lockout.RecordFailure(r.RemoteAddr)
			if isBadAuth(cfg, reqData) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				writeNicToken(w, http.StatusUnauthorized, nicTokenBadAuth)
				return
			}
			writeNicToken(w, http.StatusOK, nicTokenNoHost)
		})
	}
}

func isBadAuth(cfg *config.Config, reqData *data.ReqData) bool {
	switch cfg.Auth.Method {
	case config.AuthMethodUsers, config.AuthMethodBoth, config.AuthMethodAny:
		return !checkUserCredentials(reqData.Username, reqData.Password, cfg.Auth.Users)
	}
	return false
}

func checkUserCredentials(username, password string, users []config.User) bool {
	if username == "" || password == "" {
		return false
	}
	matched := 0
	for _, user := range users {
		matched |= constantTimeEqual(user.Username, username) & constantTimeEqual(user.Password, password)
	}
	return matched == 1
}

func NicUpdate(updater func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		inner := updater(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inner.ServeHTTP(&nicErrorWriter{
				ResponseWriter: w,
				mapStatus: func(int) (int, string) {
					return http.StatusOK, nicTokenDNSErr
				},
			}, r)
		})
	}
}

func StatusOkNicUpdate(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqData, err := data.ReqDataFromContext(r.Context())
		if err != nil {
			log.Printf("%v", err)
			writeNicToken(w, http.StatusOK, nicTokenDNSErr)
			return
		}
		writeNicToken(w, http.StatusOK, nicTokenGood+" "+reqData.Value)
	})
}

func writeNicToken(w http.ResponseWriter, status int, token string) {
	w.Header().Set(headerContentType, textPlainUTF8)
	w.WriteHeader(status)
	if _, err := fmt.Fprint(w, token); err != nil {
		log.Printf(failedWriteResponseFmt, err)
	}
}

type nicErrorWriter struct {
	http.ResponseWriter
	mapStatus func(code int) (int, string)
	handled   bool
}

func (w *nicErrorWriter) WriteHeader(code int) {
	if w.handled {
		return
	}
	if code == http.StatusOK {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.handled = true
	status, token := w.mapStatus(code)
	w.ResponseWriter.Header().Set(headerContentType, textPlainUTF8)
	w.ResponseWriter.WriteHeader(status)
	if _, err := fmt.Fprint(w.ResponseWriter, token); err != nil {
		log.Printf(failedWriteResponseFmt, err)
	}
}

func (w *nicErrorWriter) Write(b []byte) (int, error) {
	if w.handled {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
