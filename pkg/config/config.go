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
	CreateCname    string         `env:"CREATE_CNAME" envDefault:""`
	CnameTTL       int            `env:"CNAME_TTL" envDefault:"86400"`
}

type AllowedDomains map[string][]*net.IPNet

func (out *AllowedDomains) UnmarshalText(text []byte) error {
	allowedDomains := AllowedDomains{}

	parts := strings.Split(string(text), ";")
	for _, part := range parts {
		allowedParts := strings.Split(part, ",")

		if len(allowedParts) != 2 {
			return errors.New("failed to parse allowed domain, length of parts != 2")
		}

		_, ipv4Net, err := net.ParseCIDR(allowedParts[1])
		if err != nil {
			return err
		}

		allowedDomains[allowedParts[0]] = append(allowedDomains[allowedParts[0]], ipv4Net)
	}

	*out = allowedDomains
	return nil
}
