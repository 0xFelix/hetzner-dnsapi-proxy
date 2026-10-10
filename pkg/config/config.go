package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

type AllowedDomains map[string][]netip.Prefix

func (out *AllowedDomains) FromString(val string) error {
	allowedDomains := AllowedDomains{}
	for part := range strings.SplitSeq(val, ";") {
		parts := strings.Split(part, ",")

		const expectedParts = 2
		if len(parts) != expectedParts {
			return errors.New("failed to parse allowed domain, length of parts != 2")
		}

		prefix, err := parsePrefix(parts[1])
		if err != nil {
			return fmt.Errorf("invalid allowed domain %q: %w", parts[1], err)
		}

		allowedDomains[parts[0]] = append(allowedDomains[parts[0]], prefix)
	}

	*out = allowedDomains
	return nil
}

func (out *AllowedDomains) UnmarshalYAML(unmarshal func(any) error) error {
	raw := map[string][]string{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	allowedDomains := AllowedDomains{}
	for domain, entries := range raw {
		for _, entry := range entries {
			prefix, err := parsePrefix(entry)
			if err != nil {
				return fmt.Errorf("invalid allowedDomains entry %q: %w", entry, err)
			}
			allowedDomains[domain] = append(allowedDomains[domain], prefix)
		}
	}

	*out = allowedDomains
	return nil
}

type Config struct {
	BaseURL              string         `yaml:"baseURL"`
	Token                string         `yaml:"token"`
	Timeout              int            `yaml:"timeout"`
	Auth                 Auth           `yaml:"auth"`
	Endpoints            Endpoints      `yaml:"endpoints"`
	RecordTTL            int            `yaml:"recordTTL"`
	ListenAddr           string         `yaml:"listenAddr"`
	TrustedProxies       []string       `yaml:"trustedProxies"`
	TrustedProxyPrefixes []netip.Prefix `yaml:"-"`
	RateLimit            RateLimit      `yaml:"rateLimit"`
	Lockout              Lockout        `yaml:"lockout"`
	Debug                bool           `yaml:"debug"`
}

type Endpoints struct {
	Plain       bool `yaml:"plain"`
	Nic         bool `yaml:"nic"`
	AcmeDNS     bool `yaml:"acmedns"`
	HTTPReq     bool `yaml:"httpreq"`
	DirectAdmin bool `yaml:"directadmin"`
}

func (e *Endpoints) Enabled() []string {
	var names []string
	if e.Plain {
		names = append(names, EndpointPlain)
	}
	if e.Nic {
		names = append(names, EndpointNic)
	}
	if e.AcmeDNS {
		names = append(names, EndpointAcmeDNS)
	}
	if e.HTTPReq {
		names = append(names, EndpointHTTPReq)
	}
	if e.DirectAdmin {
		names = append(names, EndpointDirectAdmin)
	}
	return names
}

func (e *Endpoints) UnmarshalYAML(unmarshal func(any) error) error {
	type raw Endpoints
	var r raw
	if err := unmarshal(&r); err != nil {
		return err
	}
	*e = Endpoints(r)
	return nil
}

type Auth struct {
	Method         string         `yaml:"method"`
	AllowedDomains AllowedDomains `yaml:"allowedDomains"`
	Users          []User         `yaml:"users"`
}

const (
	EndpointPlain       = "plain"
	EndpointNic         = "nic"
	EndpointAcmeDNS     = "acmedns"
	EndpointHTTPReq     = "httpreq"
	EndpointDirectAdmin = "directadmin"
)

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
	RPS         float64 `yaml:"rps"`
	Burst       int     `yaml:"burst"`
	IdleSeconds int     `yaml:"idleSeconds"`
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
		Endpoints: Endpoints{
			Plain:       true,
			Nic:         true,
			AcmeDNS:     true,
			HTTPReq:     true,
			DirectAdmin: true,
		},
		RecordTTL:  60,
		ListenAddr: ":8081",
		RateLimit: RateLimit{
			RPS:         5,
			Burst:       10,
			IdleSeconds: 600,
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

	if err := envParse("API_TIMEOUT", &cfg.Timeout, strconv.Atoi); err != nil {
		return nil, err
	}

	allowedDomains, ok := os.LookupEnv("ALLOWED_DOMAINS")
	if !ok {
		return nil, errors.New("ALLOWED_DOMAINS environment variable not set")
	}
	if err := cfg.Auth.AllowedDomains.FromString(allowedDomains); err != nil {
		return nil, fmt.Errorf("failed to parse ALLOWED_DOMAINS: %v", err)
	}

	if err := envParse("RECORD_TTL", &cfg.RecordTTL, strconv.Atoi); err != nil {
		return nil, err
	}

	envString("LISTEN_ADDR", &cfg.ListenAddr)
	envTrustedProxies(cfg)

	if err := envParse("DEBUG", &cfg.Debug, strconv.ParseBool); err != nil {
		return nil, err
	}
	if err := envRateLimit(&cfg.RateLimit); err != nil {
		return nil, err
	}
	if err := envLockout(&cfg.Lockout); err != nil {
		return nil, err
	}
	if err := envEndpoints(&cfg.Endpoints); err != nil {
		return nil, err
	}

	prefixes, parseErr := parseTrustedProxies(cfg.TrustedProxies)
	if parseErr != nil {
		return nil, parseErr
	}
	cfg.TrustedProxyPrefixes = prefixes

	if err := validate(cfg); err != nil {
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

func envParse[T any](key string, dst *T, parse func(string) (T, error)) error {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	parsed, err := parse(v)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", key, err)
	}
	*dst = parsed
	return nil
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func envRateLimit(rl *RateLimit) error {
	return errors.Join(
		envParse("RATE_LIMIT_RPS", &rl.RPS, parseFloat),
		envParse("RATE_LIMIT_BURST", &rl.Burst, strconv.Atoi),
		envParse("RATE_LIMIT_IDLE_SECONDS", &rl.IdleSeconds, strconv.Atoi),
	)
}

func envLockout(l *Lockout) error {
	return errors.Join(
		envParse("LOCKOUT_MAX_ATTEMPTS", &l.MaxAttempts, strconv.Atoi),
		envParse("LOCKOUT_DURATION_SECONDS", &l.DurationSeconds, strconv.Atoi),
		envParse("LOCKOUT_WINDOW_SECONDS", &l.WindowSeconds, strconv.Atoi),
	)
}

func envEndpoints(endpoints *Endpoints) error {
	v, ok := os.LookupEnv("ENDPOINTS")
	if !ok {
		return nil
	}
	*endpoints = Endpoints{}
	for name := range strings.SplitSeq(v, ",") {
		name = strings.TrimSpace(name)
		switch name {
		case EndpointPlain:
			endpoints.Plain = true
		case EndpointNic:
			endpoints.Nic = true
		case EndpointAcmeDNS:
			endpoints.AcmeDNS = true
		case EndpointHTTPReq:
			endpoints.HTTPReq = true
		case EndpointDirectAdmin:
			endpoints.DirectAdmin = true
		default:
			return fmt.Errorf("invalid endpoint %q in ENDPOINTS", name)
		}
	}
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

	prefixes, parseErr := parseTrustedProxies(cfg.TrustedProxies)
	if parseErr != nil {
		return nil, parseErr
	}
	cfg.TrustedProxyPrefixes = prefixes

	if err := validate(cfg); err != nil {
		return nil, err
	}

	setDefaultBaseURL(cfg)

	return cfg, nil
}

func validate(cfg *Config) error {
	if err := validateRateLimit(&cfg.RateLimit); err != nil {
		return err
	}
	if err := validateLockout(&cfg.Lockout); err != nil {
		return err
	}
	return validateAuth(&cfg.Auth)
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
	if rl.IdleSeconds <= 0 {
		return errors.New("rateLimit.idleSeconds must be > 0")
	}
	return nil
}

func parseTrustedProxies(proxies []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(proxies))
	for _, p := range proxies {
		prefix, err := parsePrefix(p)
		if err != nil {
			return nil, fmt.Errorf("invalid trustedProxies entry %q: %w", p, err)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

func parsePrefix(s string) (netip.Prefix, error) {
	if prefix, err := netip.ParsePrefix(s); err == nil {
		return prefix.Masked(), nil
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("must be an IP address or CIDR range: %w", err)
	}
	return netip.PrefixFrom(addr, addr.BitLen()), nil
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
