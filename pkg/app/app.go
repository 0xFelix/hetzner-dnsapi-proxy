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

// LoggingResponseWriter is an exported wrapper around http.ResponseWriter to capture status code.
type LoggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int // Exported for testing
}

// WriteHeader captures the status code and calls the underlying WriteHeader.
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// NewLoggingResponseWriter creates a new LoggingResponseWriter. Exported for testing.
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
}

func New(cfg *config.Config) http.Handler {
	authorizer := middleware.NewAuthorizer(cfg)
	updater := middleware.NewUpdater(cfg)

	mux := http.NewServeMux()
	mux.Handle("GET /plain/update",
		handle(cfg, middleware.BindPlain, authorizer, updater, middleware.StatusOk))
	mux.Handle("POST /acmedns/update",
		handle(cfg, middleware.BindAcmeDNS, authorizer, updater, middleware.StatusOkAcmeDNS))
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
		lrw := NewLoggingResponseWriter(w) // Use constructor
		chain(handlers).ServeHTTP(lrw, r)
		LogRequest(r, start, lrw.StatusCode) // Use exported LogRequest
	})
}

func chain(handlers []func(http.Handler) http.Handler) http.Handler {
	if len(handlers) == 0 {
		return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	}
	return handlers[0](chain(handlers[1:]))
}

// LogRequest is an exported function to log request details.
func LogRequest(r *http.Request, start time.Time, statusCode int) {
	const methodWidth = 8
	methodPadding := strings.Repeat(" ", methodWidth-len(r.Method))
	log.Printf(
		"| %d | %13v | %15s | %s \"%s\"",
		statusCode, time.Since(start), r.RemoteAddr, r.Method+methodPadding, r.URL,
	)
}
