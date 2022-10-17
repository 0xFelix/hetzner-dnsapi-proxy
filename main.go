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

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/status"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/update"

	"github.com/caarlos0/env/v6"
	"github.com/gin-gonic/gin"
)

func startServer(listenAddr string, r *gin.Engine) {
	s := &http.Server{
		Addr:    listenAddr,
		Handler: r,
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

	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(c); err != nil {
		log.Fatal("Forcing shutdown:", err)
	}
}

func main() {
	cfg := &config.Config{}
	if err := env.Parse(cfg, env.Options{RequiredIfNoDef: true}); err != nil {
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
	r.GET("/plain/update", buildChain(cfg, data.BindPlain(), c.CheckPermissions(), c.UpdateDns(), status.Ok)...)
	r.POST("/acmedns/update", buildChain(cfg, data.BindAcmeDns(), c.CheckPermissions(), c.UpdateDns(), status.OkAcmeDns)...)
	r.POST("/acmedns/register", buildChain(cfg, status.Ok)...)
	r.POST("/httpreq/present", buildChain(cfg, data.BindHttpReq(), c.CheckPermissions(), c.UpdateDns(), status.Ok)...)
	r.POST("/httpreq/cleanup", buildChain(cfg, data.BindHttpReq(), c.CheckPermissions(), c.CleanDns(), status.Ok)...)

	log.Printf("Starting hetzner-dnsapi-proxy, listening on %s\n", cfg.ListenAddr)
	startServer(cfg.ListenAddr, r)
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
