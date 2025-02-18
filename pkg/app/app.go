package app

import (
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"
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
	authorizer := middleware.NewAuthorizer(cfg)
	updater := middleware.NewUpdater(cfg)

	mux := http.NewServeMux()
	mux.Handle("GET /plain/update",
		handle(cfg, middleware.BindPlain, authorizer, updater, middleware.StatusOk))
	mux.Handle("POST /acmedns/update",
		handle(cfg, middleware.ContentTypeJSON, middleware.BindAcmeDNS, authorizer, updater, middleware.StatusOkAcmeDNS))
	mux.Handle("POST /httpreq/present",
		handle(cfg, middleware.ContentTypeJSON, middleware.BindHTTPReq, authorizer, updater, middleware.StatusOk))
	mux.Handle("POST /httpreq/cleanup",
		handle(cfg, middleware.StatusOk))
	mux.Handle("GET /directadmin/CMD_API_SHOW_DOMAINS",
		handle(cfg, middleware.NewShowDomainsDirectAdmin(cfg)))
	mux.Handle("GET /directadmin/CMD_API_DOMAIN_POINTER",
		handle(cfg, middleware.StatusOk))
	mux.Handle("GET /directadmin/CMD_API_DNS_CONTROL",
		handle(cfg, middleware.BindDirectAdmin, authorizer, updater, middleware.StatusOkDirectAdmin))

	return mux
}

func handle(cfg *config.Config, handlers ...func(http.Handler) http.Handler) http.Handler {
	handlers = slices.Insert(handlers, 0, middleware.NewSetClientIP(cfg.TrustedProxies))
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
	methodPadding := strings.Repeat(" ", methodWidth-len(r.Method))
	log.Printf(
		"| %d | %13v | %15s | %s \"%s\"",
		statusCode, time.Since(start), r.RemoteAddr, r.Method+methodPadding, r.URL,
	)
}
