package libserver

import (
	"crypto/rand"
	"math/big"
	"net"
	"net/http/httptest"

	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/app"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func New(url string, ttl int, cloudAPI bool) (server *httptest.Server, token, username, password string) {
	const randLength = 10
	token = randString(randLength)
	username = randString(randLength)
	password = randString(randLength)

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
		RecordTTL: ttl,
		CloudAPI:  cloudAPI,
	}

	return httptest.NewServer(app.New(cfg)), token, username, password
}

func NewNoAllowedDomains(url string, cloudAPI bool) *httptest.Server {
	cfg := &config.Config{
		BaseURL: url + "/v1",
		Auth: config.Auth{
			Method: config.AuthMethodAllowedDomains,
		},
		CloudAPI: cloudAPI,
	}
	return httptest.NewServer(app.New(cfg))
}

func randString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)
	for i := range s {
		b, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		Expect(err).ToNot(HaveOccurred())
		s[i] = letters[b.Int64()]
	}
	return string(s)
}
