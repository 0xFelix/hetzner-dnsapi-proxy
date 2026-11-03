package middleware

import (
	"log"
	"net/http"
	"slices"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

// NewRecordTypeAllowed returns a middleware that rejects requests for record
// types not listed in cfg.RecordTypes with 403. It must run after the
// authorizer, so that unauthenticated clients get 401 and cannot find out
// which types are allowed.
func NewRecordTypeAllowed(cfg *config.Config) func(http.Handler) http.Handler {
	allowed := cfg.RecordTypes
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqData, err := data.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !slices.Contains(allowed, reqData.Type) {
				logTypeNotAllowed(r.RemoteAddr, reqData)
				http.Error(w, "record type not allowed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func logTypeNotAllowed(remoteAddr string, reqData *data.ReqData) {
	addr := sanitize.LogValue(remoteAddr)
	typ := sanitize.LogValue(reqData.Type)
	name := sanitize.LogValue(reqData.FullName)
	//nolint:gosec // values are sanitized above
	log.Printf("client '%s' is not allowed to set '%s' records of '%s'", addr, typ, name)
}
