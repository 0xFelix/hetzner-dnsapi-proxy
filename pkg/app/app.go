package app

import (
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/clean"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware/update"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/ratelimit"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func New(cfg *config.Config) http.Handler {
	lockout := ratelimit.NewLockout(
		cfg.Lockout.MaxAttempts,
		time.Duration(cfg.Lockout.DurationSeconds)*time.Second,
		time.Duration(cfg.Lockout.WindowSeconds)*time.Second,
	)
	authorizer := middleware.NewAuthorizer(cfg, lockout)

	m := &sync.Mutex{}
	updater := update.New(cfg, m)
	cleaner := clean.New(cfg, m)

	limiter := ratelimit.NewLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst, time.Duration(cfg.RateLimit.IdleSeconds)*time.Second)
	rl := middleware.NewRateLimit(limiter, middleware.RateLimitExceeded)

	mux := http.NewServeMux()
	mux.Handle("GET /plain/update",
		handle(cfg, rl, middleware.BindPlain, authorizer, updater, middleware.StatusOk))
	mux.Handle("GET /nic/update", handle(
		cfg, middleware.NewRateLimit(limiter, middleware.NicRateLimitExceeded), middleware.BindNicUpdate,
		middleware.NicAuth(cfg, lockout), middleware.NicUpdate(updater), middleware.StatusOkNicUpdate,
	))
	mux.Handle("POST /acmedns/update",
		handle(cfg, rl, middleware.BindAcmeDNS, authorizer, updater, middleware.StatusOkAcmeDNS))
	mux.Handle("POST /httpreq/present",
		handle(cfg, rl, middleware.ContentTypeJSON, middleware.BindHTTPReq, authorizer, updater, middleware.StatusOk))
	mux.Handle("POST /httpreq/cleanup",
		handle(cfg, rl, middleware.ContentTypeJSON, middleware.BindHTTPReq, authorizer, cleaner, middleware.StatusOk))
	mux.Handle("GET /directadmin/CMD_API_SHOW_DOMAINS",
		handle(cfg, rl, middleware.NewShowDomainsDirectAdmin(cfg, lockout)))
	mux.Handle("GET /directadmin/CMD_API_DOMAIN_POINTER",
		handle(cfg, rl, middleware.StatusOk))
	mux.Handle("GET /directadmin/CMD_API_DNS_CONTROL",
		handle(cfg, rl, middleware.BindDirectAdmin, authorizer, updater, middleware.StatusOkDirectAdmin))

	return mux
}

func handle(cfg *config.Config, handlers ...func(http.Handler) http.Handler) http.Handler {
	handlers = slices.Insert(handlers, 0, middleware.NewSetClientIP(cfg.TrustedProxyPrefixes))
	if cfg.Debug {
		handlers = slices.Insert(handlers, 0, middleware.LogDebug)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		chain(handlers).ServeHTTP(lrw, r)
		logRequest(r, start, lrw.statusCode)
	})
}

func chain(handlers []func(http.Handler) http.Handler) http.Handler {
	if len(handlers) == 0 {
		return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	}
	return handlers[0](chain(handlers[1:]))
}

func logRequest(r *http.Request, start time.Time, statusCode int) {
	const methodWidth = 8
	addr := sanitize.LogValue(r.RemoteAddr)
	method := sanitize.LogValue(r.Method)
	methodPadding := strings.Repeat(" ", max(0, methodWidth-len(method)))
	url := sanitize.LogValue(r.URL.String())
	//nolint:gosec // values are sanitized above
	log.Printf("| %d | %13v | %15s | %s \"%s\"", statusCode, time.Since(start), addr, method+methodPadding, url)
}
