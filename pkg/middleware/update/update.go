package update

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/update/cloud"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/update/dns"
)

type updater interface {
	Update(context.Context, *data.ReqData) error
}

func New(cfg *config.Config, m *sync.Mutex) func(http.Handler) http.Handler {
	var u updater
	if cfg.CloudAPI {
		u = cloud.New(cfg, m)
	} else {
		u = dns.New(cfg, m)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqData, err := data.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), hetzner.RequestTimeout)
			defer cancel()

			log.Printf("received request to update '%s' data of '%s' to '%s'", reqData.Type, reqData.FullName, reqData.Value)
			if err := u.Update(ctx, reqData); err != nil {
				log.Printf("failed to update record: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
