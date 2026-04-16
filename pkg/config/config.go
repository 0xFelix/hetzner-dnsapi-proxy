package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

type AllowedDomains map[string][]*net.IPNet

func (out *AllowedDomains) FromString(val string) error {
	allowedDomains := AllowedDomains{}
	for _, part := range strings.Split(val, ";") {
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
	BaseURL        string    `yaml:"baseURL"`
	Token          string    `yaml:"token"`
	Timeout        int       `yaml:"timeout"`
	Auth           Auth      `yaml:"auth"`
	RecordTTL      int       `yaml:"recordTTL"`
	ListenAddr     string    `yaml:"listenAddr"`
	TrustedProxies []string  `yaml:"trustedProxies"`
	RateLimit      RateLimit `yaml:"rateLimit"`
	Lockout        Lockout   `yaml:"lockout"`
	Debug          bool      `yaml:"debug"`
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

type RateLimit struct {
	RPS   float64 `yaml:"rps"`
	Burst int     `yaml:"burst"`
}

type Lockout struct {
	MaxAttempts     int `yaml:"maxAttempts"`
	DurationSeconds int `yaml:"durationSeconds"`
	WindowSeconds   int `yaml:"windowSeconds"`
}

func NewConfig() *Config {
	return &Config{
		Timeout: 60,
		Auth: Auth{
			Method: AuthMethodBoth,
		},
		RecordTTL:  60,
		ListenAddr: ":8081",
		RateLimit: RateLimit{
			RPS:   5,
			Burst: 10,
		},
		Lockout: Lockout{
			MaxAttempts:     10,
			DurationSeconds: 3600,
			WindowSeconds:   900,
		},
		Debug: false,
	}
}

func ParseEnv() (*Config, error) {
	cfg := NewConfig()
	cfg.Auth.Method = AuthMethodAllowedDomains

	envString("API_BASE_URL", &cfg.BaseURL)

	token, ok := os.LookupEnv("API_TOKEN")
	if !ok {
		return nil, errors.New("API_TOKEN environment variable not set")
	}
	cfg.Token = token
	if err := os.Unsetenv("API_TOKEN"); err != nil {
		return nil, fmt.Errorf("failed to unset API_TOKEN: %v", err)
	}

	if err := envInt("API_TIMEOUT", &cfg.Timeout); err != nil {
		return nil, err
	}

	allowedDomains, ok := os.LookupEnv("ALLOWED_DOMAINS")
	if !ok {
		return nil, errors.New("ALLOWED_DOMAINS environment variable not set")
	}
	if err := cfg.Auth.AllowedDomains.FromString(allowedDomains); err != nil {
		return nil, fmt.Errorf("failed to parse ALLOWED_DOMAINS: %v", err)
	}

	if err := envInt("RECORD_TTL", &cfg.RecordTTL); err != nil {
		return nil, err
	}

	envString("LISTEN_ADDR", &cfg.ListenAddr)
	envTrustedProxies(cfg)

	if err := envBool("DEBUG", &cfg.Debug); err != nil {
		return nil, err
	}
	if err := envFloat("RATE_LIMIT_RPS", &cfg.RateLimit.RPS); err != nil {
		return nil, err
	}
	if err := envInt("RATE_LIMIT_BURST", &cfg.RateLimit.Burst); err != nil {
		return nil, err
	}
	if err := envInt("LOCKOUT_MAX_ATTEMPTS", &cfg.Lockout.MaxAttempts); err != nil {
		return nil, err
	}
	if err := envInt("LOCKOUT_DURATION_SECONDS", &cfg.Lockout.DurationSeconds); err != nil {
		return nil, err
	}
	if err := envInt("LOCKOUT_WINDOW_SECONDS", &cfg.Lockout.WindowSeconds); err != nil {
		return nil, err
	}

	setDefaultBaseURL(cfg)

	return cfg, nil
}

func envString(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func envInt(key string, dst *int) error {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", key, err)
	}
	*dst = i
	return nil
}

func envFloat(key string, dst *float64) error {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", key, err)
	}
	*dst = f
	return nil
}

func envBool(key string, dst *bool) error {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", key, err)
	}
	*dst = b
	return nil
}

func envTrustedProxies(cfg *Config) {
	v, ok := os.LookupEnv("TRUSTED_PROXIES")
	if !ok {
		return
	}
	cfg.TrustedProxies = strings.Split(v, ",")
	for i := range cfg.TrustedProxies {
		cfg.TrustedProxies[i] = strings.TrimSpace(cfg.TrustedProxies[i])
	}
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

	if err := validateRateLimit(&cfg.RateLimit); err != nil {
		return nil, err
	}
	if err := validateLockout(&cfg.Lockout); err != nil {
		return nil, err
	}
	if err := validateAuth(&cfg.Auth); err != nil {
		return nil, err
	}

	setDefaultIPMask(cfg.Auth.AllowedDomains)
	setDefaultBaseURL(cfg)

	return cfg, nil
}

func validateAuth(a *Auth) error {
	if !AuthMethodIsValid(a.Method) {
		return fmt.Errorf("invalid auth method: %s", a.Method)
	}
	if len(a.AllowedDomains) == 0 && (a.Method == AuthMethodAllowedDomains || a.Method == AuthMethodBoth) {
		return fmt.Errorf("auth.allowedDomains cannot be empty with auth method %s", a.Method)
	}
	if len(a.Users) == 0 && (a.Method == AuthMethodUsers || a.Method == AuthMethodBoth) {
		return fmt.Errorf("auth.users cannot be empty with auth method %s", a.Method)
	}
	if len(a.AllowedDomains) == 0 && len(a.Users) == 0 && a.Method == AuthMethodAny {
		return errors.New("auth.allowedDomains or auth.users cannot both be empty with auth method any")
	}
	return nil
}

func validateRateLimit(rl *RateLimit) error {
	if rl.RPS <= 0 {
		return errors.New("rateLimit.rps must be > 0")
	}
	if rl.Burst <= 0 {
		return errors.New("rateLimit.burst must be > 0")
	}
	return nil
}

func validateLockout(l *Lockout) error {
	if l.MaxAttempts <= 0 {
		return errors.New("lockout.maxAttempts must be > 0")
	}
	if l.DurationSeconds <= 0 {
		return errors.New("lockout.durationSeconds must be > 0")
	}
	if l.WindowSeconds <= 0 {
		return errors.New("lockout.windowSeconds must be > 0")
	}
	return nil
}

func AuthMethodIsValid(authMethod string) bool {
	return authMethod == AuthMethodAllowedDomains ||
		authMethod == AuthMethodUsers ||
		authMethod == AuthMethodBoth ||
		authMethod == AuthMethodAny
}

func setDefaultBaseURL(c *Config) {
	if c.BaseURL == "" {
		c.BaseURL = "https://api.hetzner.cloud/v1"
	}
}

func setDefaultIPMask(allowedDomains AllowedDomains) {
	const ff = 255
	for _, allowedDomain := range allowedDomains {
		for _, ipNet := range allowedDomain {
			if len(ipNet.Mask) == 0 {
				ipNet.Mask = net.IPv4Mask(ff, ff, ff, ff)
			}
		}
	}
}
