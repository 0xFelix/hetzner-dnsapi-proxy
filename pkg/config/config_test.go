package config_test

import (
	"net"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

var _ = Describe("AllowedDomains", func() {
	const unexpectedPartsCountErr = "failed to parse allowed domain, length of parts != 2"

	DescribeTable("should unmarshal text successfully", func(text string, expected func() config.AllowedDomains) {
		allowedDomains := config.AllowedDomains{}
		Expect(allowedDomains.UnmarshalText([]byte(text))).To(Succeed())
		Expect(allowedDomains).To(Equal(expected()))
	},
		Entry("wildcard for localhost", "*,127.0.0.1/32",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("wildcard for remote host", "*,192.168.0.0/16",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"*": []*net.IPNet{ipNet}}
			},
		),
		Entry("domain for host", "example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{"example.com": []*net.IPNet{ipNet}}
			},
		),
		Entry("two entries", "*,127.0.0.1/32;example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"*":           []*net.IPNet{ipNetLocalhost},
					"example.com": []*net.IPNet{ipNetRemote},
				}
			},
		),
		Entry("three entries", "*,127.0.0.1/32;example.com,192.168.0.1/32;test.com,127.0.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"*":           []*net.IPNet{ipNetLocalhost},
					"example.com": []*net.IPNet{ipNetRemote},
					"test.com":    []*net.IPNet{ipNetLocalhost},
				}
			},
		),
		Entry("multiple entries for same domain", "example.com,127.0.0.1/32;example.com,192.168.0.1/32",
			func() config.AllowedDomains {
				_, ipNetLocalhost, err := net.ParseCIDR("127.0.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				_, ipNetRemote, err := net.ParseCIDR("192.168.0.1/32")
				Expect(err).NotTo(HaveOccurred())
				return config.AllowedDomains{
					"example.com": []*net.IPNet{ipNetLocalhost, ipNetRemote},
				}
			},
		),
	)

	DescribeTable("should fail unmarshal text on", func(text, expected string) {
		allowedDomains := config.AllowedDomains{}
		Expect(allowedDomains.UnmarshalText([]byte(text))).To(MatchError(expected))
		Expect(allowedDomains).To(BeEmpty())
	},
		Entry("empty", "", unexpectedPartsCountErr),
		Entry("empty after entry", "*,127.0.0.1/32;", unexpectedPartsCountErr),
		Entry("empty before entry", ";*,127.0.0.1/32", unexpectedPartsCountErr),
		Entry("empty between entries", "*,127.0.0.1/32;;*,127.0.0.1/32", unexpectedPartsCountErr),
		Entry("invalid CIDR", "*,127.0.0.1/64;", "invalid CIDR address: 127.0.0.1/64"),
	)
})

var _ = Describe("Config", func() {
	const (
		baseURL           = "https://test.com"
		apiToken          = "verysecrettoken"
		apiTimeout        = 1200
		apiTimeoutStr     = "1200"
		allowedDomainsStr = "*,127.0.0.1/32"
		recordTTL         = 5000
		recordTTLStr      = "5000"
		listenAddr        = "127.0.0.1:8080"
		trustedProxiesStr = "127.0.0.1,192.168.0.1,192.168.0.2"
		debug             = true
		debugStr          = "true"
	)

	var (
		allowedDomains config.AllowedDomains
		trustedProxies []string
	)

	BeforeEach(func() {
		trustedProxies = []string{"127.0.0.1", "192.168.0.1", "192.168.0.2"}
	})

	Context("ParseEnv", func() {
		const (
			envAPIBaseURL     = "API_BASE_URL"
			envAPIToken       = "API_TOKEN"
			envAPITimeout     = "API_TIMEOUT"
			envAllowedDomains = "ALLOWED_DOMAINS"
			envRecordTTL      = "RECORD_TTL"
			envListenAddr     = "LISTEN_ADDR"
			envTrustedProxies = "TRUSTED_PROXIES"
			envDebug          = "DEBUG"
		)

		BeforeEach(func() {
			allowedDomains = config.AllowedDomains{
				"*": []*net.IPNet{
					{
						IP:   net.IPv4(127, 0, 0, 1).To4(),
						Mask: net.IPv4Mask(255, 255, 255, 255),
					},
				},
			}
		})

		AfterEach(func() {
			Expect(os.Unsetenv(envAPIBaseURL)).To(Succeed())
			Expect(os.Unsetenv(envAPIToken)).To(Succeed())
			Expect(os.Unsetenv(envAPITimeout)).To(Succeed())
			Expect(os.Unsetenv(envAllowedDomains)).To(Succeed())
			Expect(os.Unsetenv(envRecordTTL)).To(Succeed())
			Expect(os.Unsetenv(envListenAddr)).To(Succeed())
			Expect(os.Unsetenv(envTrustedProxies)).To(Succeed())
			Expect(os.Unsetenv(envDebug)).To(Succeed())
		})

		It("should parse environment successfully", func() {
			Expect(os.Setenv(envAPIBaseURL, baseURL)).To(Succeed())
			Expect(os.Setenv(envAPIToken, apiToken)).To(Succeed())
			Expect(os.Setenv(envAPITimeout, apiTimeoutStr)).To(Succeed())
			Expect(os.Setenv(envAllowedDomains, allowedDomainsStr)).To(Succeed())
			Expect(os.Setenv(envRecordTTL, recordTTLStr)).To(Succeed())
			Expect(os.Setenv(envListenAddr, listenAddr)).To(Succeed())
			Expect(os.Setenv(envTrustedProxies, trustedProxiesStr)).To(Succeed())
			Expect(os.Setenv(envDebug, debugStr)).To(Succeed())

			cfg, err := config.ParseEnv()
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.BaseURL).To(Equal(baseURL))
			Expect(cfg.Token).To(Equal(apiToken))
			Expect(cfg.Timeout).To(Equal(apiTimeout))
			Expect(cfg.Auth.Method).To(Equal(config.AuthMethodAllowedDomains))
			Expect(cfg.Auth.AllowedDomains).To(Equal(allowedDomains))
			Expect(cfg.Auth.Users).To(BeEmpty())
			Expect(cfg.RecordTTL).To(Equal(recordTTL))
			Expect(cfg.ListenAddr).To(Equal(listenAddr))
			Expect(cfg.TrustedProxies).To(Equal(trustedProxies))
			Expect(cfg.Debug).To(Equal(debug))
		})

		DescribeTable("should fail on invalid environment variables", func(setEnv func(), errMsg string) {
			setEnv()
			cfg, err := config.ParseEnv()
			Expect(err).To(MatchError(errMsg))
			Expect(cfg).To(BeNil())
		},
			Entry("API_TOKEN missing", func() {}, "API_TOKEN environment variable not set"),
			Entry("ALLOWED_DOMAINS missing", func() {
				Expect(os.Setenv(envAPIToken, apiToken)).To(Succeed())
			}, "ALLOWED_DOMAINS environment variable not set"),
			Entry("API_TIMEOUT not an int", func() {
				Expect(os.Setenv(envAPIToken, apiToken)).To(Succeed())
				Expect(os.Setenv(envAPITimeout, "something")).To(Succeed())
			}, "failed to parse API_TIMEOUT: strconv.Atoi: parsing \"something\": invalid syntax"),
			Entry("RECORD_TTL not an int", func() {
				Expect(os.Setenv(envAPIToken, apiToken)).To(Succeed())
				Expect(os.Setenv(envAllowedDomains, allowedDomainsStr)).To(Succeed())
				Expect(os.Setenv(envRecordTTL, "something")).To(Succeed())
			}, "failed to parse RECORD_TTL: strconv.Atoi: parsing \"something\": invalid syntax"),
			Entry("DEBUG not a bool", func() {
				Expect(os.Setenv(envAPIToken, apiToken)).To(Succeed())
				Expect(os.Setenv(envAllowedDomains, allowedDomainsStr)).To(Succeed())
				Expect(os.Setenv(envDebug, "something")).To(Succeed())
			}, "failed to parse DEBUG: strconv.ParseBool: parsing \"something\": invalid syntax"),
		)
	})

	Context("ReadFile", func() {
		var (
			users    []config.User
			filePath string
		)

		BeforeEach(func() {
			allowedDomains = config.AllowedDomains{
				"*": []*net.IPNet{
					{
						IP:   net.IPv4(127, 0, 0, 1),
						Mask: net.IPv4Mask(255, 255, 255, 255),
					},
				},
			}

			users = []config.User{
				{
					Username: "testname",
					Password: "testpassword",
					Domains:  []string{"test.tld"},
				},
			}

			filePath = path.Join(GinkgoT().TempDir(), "config.yaml")
		})

		It("should read file successfully", func() {
			cfg := &config.Config{
				BaseURL: baseURL,
				Token:   apiToken,
				Timeout: apiTimeout,
				Auth: config.Auth{
					Method:         config.AuthMethodBoth,
					AllowedDomains: allowedDomains,
					Users:          users,
				},
				RecordTTL:      recordTTL,
				ListenAddr:     listenAddr,
				TrustedProxies: trustedProxies,
				Debug:          debug,
			}

			data, err := yaml.Marshal(cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(filePath, data, 0o600)).To(Succeed())

			cfgRead, err := config.ReadFile(filePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfgRead).To(Equal(cfg))
		})

		DescribeTable("should fail on ", func(cfgFn func() *config.Config, errMsg string) {
			data, err := yaml.Marshal(cfgFn())
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(filePath, data, 0o600)).To(Succeed())

			cfgRead, err := config.ReadFile(filePath)
			Expect(err).To(MatchError(errMsg))
			Expect(cfgRead).To(BeNil())
		},
			Entry("missing token", func() *config.Config { return &config.Config{} }, "token is required"),
			Entry("invalid auth method",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method:         "something",
							AllowedDomains: allowedDomains,
							Users:          users,
						},
					}
				},
				"invalid auth method: something",
			),
			Entry("empty allowed domains with auth method allowedDomains",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method: config.AuthMethodAllowedDomains,
							Users:  users,
						},
					}
				},
				"auth.allowedDomains cannot be empty with auth method allowedDomains",
			),
			Entry("empty allowed domains with auth method both",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method: config.AuthMethodBoth,
							Users:  users,
						},
					}
				},
				"auth.allowedDomains cannot be empty with auth method both",
			),
			Entry("empty users with auth method users",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method:         config.AuthMethodUsers,
							AllowedDomains: allowedDomains,
						},
					}
				},
				"auth.users cannot be empty with auth method users",
			),
			Entry("empty users with auth method both",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method:         config.AuthMethodBoth,
							AllowedDomains: allowedDomains,
						},
					}
				},
				"auth.users cannot be empty with auth method both",
			),
			Entry("empty allowed domains and users with auth method any",
				func() *config.Config {
					return &config.Config{
						Token: apiToken,
						Auth: config.Auth{
							Method: config.AuthMethodAny,
						},
					}
				},
				"auth.allowedDomains or auth.users cannot both be empty with auth method any",
			),
		)

		It("should fail on invalid yaml", func() {
			Expect(os.WriteFile(filePath, []byte("not yaml"), 0o600)).To(Succeed())
			cfg, err := config.ReadFile(filePath)
			Expect(err).To(MatchError(ContainSubstring("yaml: unmarshal errors:")))
			Expect(cfg).To(BeNil())
		})
	})
})
