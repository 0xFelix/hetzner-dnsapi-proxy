package config

import (
	"errors"
	"net"
	"strings"
)

type Config struct {
	Token          string         `env:"API_TOKEN,unset"`
	Timeout        int            `env:"API_TIMEOUT" envDefault:"15"`
	AllowedDomains AllowedDomains `env:"ALLOWED_DOMAINS"`
	RecordTTL      int            `env:"RECORD_TTL" envDefault:"60"`
	ListenAddr     string         `env:"LISTEN_ADDR" envDefault:":8081"`
	TrustedProxies []string       `env:"TRUSTED_PROXIES" envDefault:""`
	Debug          bool           `env:"DEBUG" envDefault:"false"`
}

type AllowedDomains map[string][]*net.IPNet

func (out *AllowedDomains) UnmarshalText(text []byte) error {
	const expectedPartsCount = 2

	allowedDomains := AllowedDomains{}
	for _, part := range strings.Split(string(text), ";") {
		partSplit := strings.Split(part, ",")

		if len(partSplit) != expectedPartsCount {
			return errors.New("failed to parse allowed domain, length of parts != 2")
		}

		_, ipv4Net, err := net.ParseCIDR(partSplit[1])
		if err != nil {
			return err
		}

		allowedDomains[partSplit[0]] = append(allowedDomains[partSplit[0]], ipv4Net)
	}

	*out = allowedDomains
	return nil
}
