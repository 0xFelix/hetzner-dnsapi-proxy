package status

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/common"
)

func Ok(c *gin.Context) {
	c.Status(http.StatusOK)
}

func OkAcmeDNS(c *gin.Context) {
	record, ok := c.MustGet(common.KeyDNSRecord).(*common.DNSRecord)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, gin.H{"txt": record.Value})
}

func OkDirectAdmin(c *gin.Context) {
	var values url.Values
	values.Set("error", "0")
	values.Set("text", "OK")
	c.Data(http.StatusOK, "application/x-www-form-urlencoded", []byte(values.Encode()))
}
