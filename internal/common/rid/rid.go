package rid

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func GetRID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := strings.TrimSpace(c.GetString("rid")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-Id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	return ""
}
