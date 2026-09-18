package libserver

import (
	"crypto/rand"
	"net"
	"net/http/httptest"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/app"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func New(url string, ttl int) (server *httptest.Server, token, username, password string) {
	token = rand.Text()
	username = rand.Text()
	password = rand.Text()

	cfg := &config.Config{
		BaseURL: url + "/v1",
		Token:   token,
		Timeout: 10,
		Auth: config.Auth{
			Method: config.AuthMethodBoth,
			AllowedDomains: config.AllowedDomains{
				"*": []*net.IPNet{{
					IP:   net.IPv4(127, 0, 0, 1),           //nolint:mnd
					Mask: net.IPv4Mask(255, 255, 255, 255), //nolint:mnd
				}},
			},
			Users: []config.User{{
				Username: username,
				Password: password,
				Domains:  []string{"*"},
			}},
		},
		Endpoints: config.Endpoints{Plain: true, Nic: true, AcmeDNS: true, HTTPReq: true, DirectAdmin: true},
		RecordTTL: ttl,
		RateLimit: config.RateLimit{RPS: 1000, Burst: 1000, IdleSeconds: 600},
		Lockout:   config.Lockout{MaxAttempts: 1000, DurationSeconds: 3600, WindowSeconds: 900},
	}

	return httptest.NewServer(app.New(cfg)), token, username, password
}

func NewNoAllowedDomains(url string) *httptest.Server {
	cfg := &config.Config{
		BaseURL: url + "/v1",
		Auth: config.Auth{
			Method: config.AuthMethodAllowedDomains,
		},
		Endpoints: config.Endpoints{Plain: true, Nic: true, AcmeDNS: true, HTTPReq: true, DirectAdmin: true},
		RateLimit: config.RateLimit{RPS: 1000, Burst: 1000, IdleSeconds: 600},
		Lockout:   config.Lockout{MaxAttempts: 1000, DurationSeconds: 3600, WindowSeconds: 900},
	}
	return httptest.NewServer(app.New(cfg))
}
