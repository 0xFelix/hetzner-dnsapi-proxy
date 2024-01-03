package common

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	ContentTypeURLEncoded = "application/x-www-form-urlencoded"
)

func StatusOkDirectAdmin(c *gin.Context) {
	values := url.Values{}
	values.Set("error", "0")
	values.Set("text", "OK")
	c.Data(http.StatusOK, ContentTypeURLEncoded, []byte(values.Encode()))
}
