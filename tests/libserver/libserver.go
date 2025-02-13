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

func New(url string, ttl int) (server *httptest.Server, token string) {
	const tokenLength = 10
	token = randString(tokenLength)

	_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
	Expect(err).ToNot(HaveOccurred())

	return httptest.NewServer(app.New(
		&config.Config{
			BaseURL: url + "/v1",
			Token:   token,
			AllowedDomains: config.AllowedDomains{
				"*": []*net.IPNet{ipNet},
			},
			RecordTTL: ttl,
		},
	)), token
}

func NewNoAllowedDomains(url string) *httptest.Server {
	return httptest.NewServer(app.New(
		&config.Config{
			BaseURL: url + "/v1",
		},
	))
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
