package clean

import (
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/handler/cleaner/cloud"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/handler/cleaner/dns"
)

func New(cfg *config.Config) func(http.Handler) http.Handler {
	if cfg.CloudAPI {
		return cloud.New(cfg)
	}

	return dns.New(cfg)
}
