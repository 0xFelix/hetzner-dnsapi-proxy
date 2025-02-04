package common

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func StatusOkDirectAdmin(c *gin.Context) {
	values := url.Values{}
	values.Set("error", "0")
	values.Set("text", "OK")
	c.Data(http.StatusOK, "application/x-www-form-urlencoded", []byte(values.Encode()))
}
