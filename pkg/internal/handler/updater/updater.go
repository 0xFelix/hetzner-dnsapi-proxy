package update

import (
	"net/http"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/handler/updater/cloud"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/handler/updater/dns"
)

func New(cfg *config.Config) func(http.Handler) http.Handler {
	if cfg.CloudAPI {
		return cloud.New(cfg)
	}

	return dns.New(cfg)
}
