package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type AllowedDomains map[string][]*net.IPNet

func (out *AllowedDomains) UnmarshalText(text []byte) error {
	allowedDomains := AllowedDomains{}
	for _, part := range strings.Split(string(text), ";") {
		parts := strings.Split(part, ",")

		const expectedParts = 2
		if len(parts) != expectedParts {
			return errors.New("failed to parse allowed domain, length of parts != 2")
		}

		_, ipNet, err := net.ParseCIDR(parts[1])
		if err != nil {
			return err
		}

		allowedDomains[parts[0]] = append(allowedDomains[parts[0]], ipNet)
	}

	*out = allowedDomains
	return nil
}

type Config struct {
	BaseURL        string
	Token          string
	Timeout        int
	AllowedDomains AllowedDomains
	RecordTTL      int
	ListenAddr     string
	TrustedProxies []string
	Debug          bool
}

func ParseEnv() (*Config, error) {
	cfg := &Config{
		BaseURL:    "https://dns.hetzner.com/api/v1",
		Timeout:    15,
		RecordTTL:  60,
		ListenAddr: ":8081",
		Debug:      false,
	}

	if baseURL, ok := os.LookupEnv("API_BASE_URL"); ok {
		cfg.BaseURL = baseURL
	}

	if token, ok := os.LookupEnv("API_TOKEN"); ok {
		cfg.Token = token
		if err := os.Unsetenv("API_TOKEN"); err != nil {
			return nil, fmt.Errorf("failed to unset API_TOKEN: %v", err)
		}
	} else {
		return nil, errors.New("API_TOKEN environment variable not set")
	}

	if timeout, ok := os.LookupEnv("API_TIMEOUT"); ok {
		timeoutInt, err := strconv.Atoi(timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse API_TIMEOUT: %v", err)
		}
		cfg.Timeout = timeoutInt
	}

	if allowedDomains, ok := os.LookupEnv("ALLOWED_DOMAINS"); ok {
		if err := cfg.AllowedDomains.UnmarshalText([]byte(allowedDomains)); err != nil {
			return nil, fmt.Errorf("failed to parse ALLOWED_DOMAINS: %v", err)
		}
	} else {
		return nil, errors.New("ALLOWED_DOMAINS environment variable not set")
	}

	if recordTTL, ok := os.LookupEnv("RECORD_TTL"); ok {
		recordTTLInt, err := strconv.Atoi(recordTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RECORD_TTL: %v", err)
		}
		cfg.RecordTTL = recordTTLInt
	}

	if listAddr, ok := os.LookupEnv("LISTEN_ADDR"); ok {
		cfg.ListenAddr = listAddr
	}

	if trustedProxies, ok := os.LookupEnv("TRUSTED_PROXIES"); ok {
		cfg.TrustedProxies = strings.Split(trustedProxies, ",")
		for i := range cfg.TrustedProxies {
			cfg.TrustedProxies[i] = strings.TrimSpace(cfg.TrustedProxies[i])
		}
	}

	if debug, ok := os.LookupEnv("DEBUG"); ok {
		debugBool, err := strconv.ParseBool(debug)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DEBUG: %v", err)
		}
		cfg.Debug = debugBool
	}

	return cfg, nil
}
