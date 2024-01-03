package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/gin-gonic/gin"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/status"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/update"
)

const (
	readHeaderTimeout = 10
	shutdownTimeout   = 5
)

func runServer(listenAddr string, r *gin.Engine) error {
	s := &http.Server{
		Addr:              listenAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout * time.Second,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down hetzner-dnsapi-proxy")

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	defer cancel()

	return s.Shutdown(c)
}

func main() {
	cfg := &config.Config{}
	if err := env.ParseWithOptions(cfg, env.Options{RequiredIfNoDef: true}); err != nil {
		log.Fatal(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			log.Fatal(err)
		}
	}

	c := update.NewController(cfg)
	r.GET("/plain/update", buildChain(cfg, data.BindPlain(), c.CheckPermissions(), c.UpdateDNS(), status.Ok)...)
	r.POST("/acmedns/update", buildChain(cfg, data.BindAcmeDNS(), c.CheckPermissions(), c.UpdateDNS(), status.OkAcmeDNS)...)
	r.POST("/acmedns/register", buildChain(cfg, status.Ok)...)
	r.POST("/httpreq/present", buildChain(cfg, data.BindHTTPReq(), c.CheckPermissions(), c.UpdateDNS(), status.Ok)...)
	r.POST("/httpreq/cleanup", buildChain(cfg, status.Ok)...)
	r.GET("/directadmin/CMD_API_SHOW_DOMAINS", buildChain(cfg, data.ShowDomainsDirectAdmin(cfg.AllowedDomains))...)
	r.GET("/directadmin/CMD_API_DNS_CONTROL",
		buildChain(cfg, data.BindDirectAdmin(), c.CheckPermissions(), c.UpdateDNS(), status.OkDirectAdmin)...)

	log.Printf("Starting hetzner-dnsapi-proxy, listening on %s\n", cfg.ListenAddr)
	if err := runServer(cfg.ListenAddr, r); err != nil {
		log.Fatal("Error running server:", err)
	}
}

func buildChain(cfg *config.Config, handlers ...gin.HandlerFunc) gin.HandlersChain {
	if cfg.Debug {
		handlers = append([]gin.HandlerFunc{RequestLoggerMiddleware()}, handlers...)
	}
	return handlers
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		body, _ := io.ReadAll(io.TeeReader(c.Request.Body, &buf))
		c.Request.Body = io.NopCloser(&buf)
		log.Printf("BODY %s", string(body))
		log.Printf("HEADER %+v", c.Request.Header)
		c.Next()
	}
}
