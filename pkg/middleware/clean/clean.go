package clean

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/clean/cloud"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/clean/dns"
)

type cleaner interface {
	Clean(context.Context, *data.ReqData) error
}

func New(cfg *config.Config, m *sync.Mutex) func(http.Handler) http.Handler {
	var c cleaner
	if cfg.CloudAPI {
		c = cloud.New(cfg, m)
	} else {
		c = dns.New()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqData, err := data.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Printf("received request to clean '%s' data of '%s'", reqData.Type, reqData.FullName)
			ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.Timeout)*time.Second)
			defer cancel()
			if err := c.Clean(ctx, reqData); err != nil {
				log.Printf("failed to clean record: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
