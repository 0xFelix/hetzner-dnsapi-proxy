package app

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/status"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/update"
)

func New(cfg *config.Config) http.Handler {
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
	r.POST("/httpreq/present", buildChain(cfg, data.BindHTTPReq(), c.CheckPermissions(), c.UpdateDNS(), status.Ok)...)
	r.POST("/httpreq/cleanup", buildChain(cfg, status.Ok)...)
	r.GET("/directadmin/CMD_API_SHOW_DOMAINS", buildChain(cfg, data.ShowDomainsDirectAdmin(cfg.AllowedDomains))...)
	r.GET("/directadmin/CMD_API_DOMAIN_POINTER", buildChain(cfg, status.Ok)...)
	r.GET("/directadmin/CMD_API_DNS_CONTROL",
		buildChain(cfg, data.BindDirectAdmin(), c.CheckPermissions(), c.UpdateDNS(), status.OkDirectAdmin)...)

	return r
}

func buildChain(cfg *config.Config, handlers ...gin.HandlerFunc) gin.HandlersChain {
	if cfg.Debug {
		handlers = append([]gin.HandlerFunc{requestLoggerMiddleware()}, handlers...)
	}
	return handlers
}

func requestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		body, _ := io.ReadAll(io.TeeReader(c.Request.Body, &buf))
		c.Request.Body = io.NopCloser(&buf)
		log.Printf("BODY %s", string(body))
		log.Printf("HEADER %+v", c.Request.Header)
		c.Next()
	}
}
