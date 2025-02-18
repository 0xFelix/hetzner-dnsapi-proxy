package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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
	BaseURL        string   `yaml:"baseURL"`
	Token          string   `yaml:"token"`
	Timeout        int      `yaml:"timeout"`
	Auth           Auth     `yaml:"auth"`
	RecordTTL      int      `yaml:"recordTTL"`
	ListenAddr     string   `yaml:"listenAddr"`
	TrustedProxies []string `yaml:"trustedProxies"`
	Debug          bool     `yaml:"debug"`
}

type Auth struct {
	Method         string         `yaml:"method"`
	AllowedDomains AllowedDomains `yaml:"allowedDomains"`
	Users          []User         `yaml:"users"`
}

const (
	AuthMethodAllowedDomains = "allowedDomains"
	AuthMethodUsers          = "users"
	AuthMethodBoth           = "both"
	AuthMethodAny            = "any"
)

type User struct {
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Domains  []string `yaml:"domains"`
}

func NewConfig() *Config {
	return &Config{
		BaseURL: "https://dns.hetzner.com/api/v1",
		Timeout: 15,
		Auth: Auth{
			Method: AuthMethodBoth,
		},
		RecordTTL:  60,
		ListenAddr: ":8081",
		Debug:      false,
	}
}

func ParseEnv() (*Config, error) {
	cfg := NewConfig()
	cfg.Auth.Method = AuthMethodAllowedDomains

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
		if err := cfg.Auth.AllowedDomains.UnmarshalText([]byte(allowedDomains)); err != nil {
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

func ReadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := NewConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Token == "" {
		return nil, errors.New("token is required")
	}

	if !AuthMethodIsValid(cfg.Auth.Method) {
		return nil, fmt.Errorf("invalid auth method: %s", cfg.Auth.Method)
	}

	if len(cfg.Auth.AllowedDomains) == 0 && (cfg.Auth.Method == AuthMethodAllowedDomains || cfg.Auth.Method == AuthMethodBoth) {
		return nil, fmt.Errorf("auth.allowedDomains cannot be empty with auth method %s", cfg.Auth.Method)
	}

	if len(cfg.Auth.Users) == 0 && (cfg.Auth.Method == AuthMethodUsers || cfg.Auth.Method == AuthMethodBoth) {
		return nil, fmt.Errorf("auth.users cannot be empty with auth method %s", cfg.Auth.Method)
	}

	if len(cfg.Auth.AllowedDomains) == 0 && len(cfg.Auth.Users) == 0 && cfg.Auth.Method == AuthMethodAny {
		return nil, errors.New("auth.allowedDomains or auth.users cannot both be empty with auth method any")
	}

	return cfg, nil
}

func AuthMethodIsValid(authMethod string) bool {
	return authMethod == AuthMethodAllowedDomains ||
		authMethod == AuthMethodUsers ||
		authMethod == AuthMethodBoth ||
		authMethod == AuthMethodAny
}
