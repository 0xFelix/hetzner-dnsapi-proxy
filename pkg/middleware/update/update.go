package update

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/update/cloud"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

func New(cfg *config.Config) func(http.Handler) http.Handler {
	u := cloud.New(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqData, err := data.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			logUpdateRequest(reqData)
			ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.Timeout)*time.Second)
			defer cancel()
			if err := u.Update(ctx, reqData); err != nil {
				log.Printf("failed to update record: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func logUpdateRequest(reqData *data.ReqData) {
	typ := sanitize.LogValue(reqData.Type)
	name := sanitize.LogValue(reqData.FullName)
	val := sanitize.LogValue(reqData.Value)
	log.Printf("received request to update '%s' data of '%s' to '%s'", typ, name, val)
}
