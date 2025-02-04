package data

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/common"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

const (
	KeyRecord = "KeyRecord"

	prefixAcmeChallenge = "_acme-challenge"
	recordTypeA         = "A"
	recordTypeTXT       = "TXT"
)

type DNSRecord struct {
	FullName string
	Name     string
	Zone     string
	Value    string
	Type     string
}

type plainData struct {
	FullName string `form:"hostname" binding:"required"`
	Value    string `form:"ip" binding:"required"`
}

type acmeDNSData struct {
	FullName string `json:"subdomain" binding:"required"`
	Value    string `json:"txt" binding:"required"`
}

type httpReqData struct {
	FullName string `json:"fqdn" binding:"required"`
	Value    string `json:"value" binding:"required"`
}

type directAdminData struct {
	Domain string `form:"domain" binding:"required"`
	Action string `form:"action" binding:"required"`
	Type   string `form:"type"`
	Name   string `form:"name"`
	Value  string `form:"value"`
}

func BindPlain() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := plainData{}

		if err := c.Bind(&data); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		name, zone := splitFullName(data.FullName)
		c.Set(KeyRecord, &DNSRecord{
			FullName: data.FullName,
			Name:     name,
			Zone:     zone,
			Value:    data.Value,
			Type:     recordTypeA,
		})
	}
}

func BindAcmeDNS() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := acmeDNSData{}

		if err := c.BindJSON(&data); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		// prepend prefix if not already given
		if !strings.HasPrefix(data.FullName, prefixAcmeChallenge) {
			data.FullName = fmt.Sprintf("%s.%s", prefixAcmeChallenge, data.FullName)
		}

		name, zone := splitFullName(data.FullName)
		c.Set(KeyRecord, &DNSRecord{
			FullName: data.FullName,
			Name:     name,
			Zone:     zone,
			Value:    data.Value,
			Type:     recordTypeTXT,
		})
	}
}

func BindHTTPReq() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := httpReqData{}

		if err := c.Bind(&data); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		data.FullName = strings.TrimRight(data.FullName, ".")
		name, zone := splitFullName(data.FullName)
		c.Set(KeyRecord, &DNSRecord{
			FullName: data.FullName,
			Name:     name,
			Zone:     zone,
			Value:    data.Value,
			Type:     recordTypeTXT,
		})
	}
}

func ShowDomainsDirectAdmin(allowedDomains config.AllowedDomains) gin.HandlerFunc {
	return func(c *gin.Context) {
		domains := map[string]struct{}{}
		for domain := range allowedDomains {
			domains[strings.TrimPrefix(domain, "*.")] = struct{}{}
		}

		values := url.Values{}
		for domain := range domains {
			values.Add("list", domain)
		}

		c.Data(http.StatusOK, common.ContentTypeURLEncoded, []byte(values.Encode()))
	}
}

func BindDirectAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := directAdminData{}

		if err := c.Bind(&data); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if data.Action != "add" {
			c.Abort()
			common.StatusOkDirectAdmin(c)
			return
		}

		if data.Type != recordTypeA && data.Type != recordTypeTXT {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		fullName := ""
		if data.Name != "" {
			fullName = data.Name + "." + data.Domain
		} else {
			fullName = data.Domain
		}

		name, zone := splitFullName(fullName)
		c.Set(KeyRecord, &DNSRecord{
			FullName: fullName,
			Name:     name,
			Zone:     zone,
			Value:    data.Value,
			Type:     data.Type,
		})
	}
}

func splitFullName(n string) (name, zone string) {
	parts := strings.Split(n, ".")
	length := len(parts)

	for i := 0; i < length-2; i++ {
		name += parts[i]
		if i < length-3 {
			name += "."
		}
	}

	zone = fmt.Sprintf("%s.%s", parts[length-2], parts[length-1])

	return
}
