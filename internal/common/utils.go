package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetStringFromCtx(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	default:
		return strings.TrimSpace(FmtAny(t))
	}
}

func FirstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func TruncateString(s string, max int) string {
	if max <= 0 {
		return s
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

// GetInt64FromCtx 从 gin.Context 读取 int64（兼容常见数字类型）。
func GetInt64FromCtx(c *gin.Context, key string) int64 {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case uint:
		return int64(t)
	case uint32:
		return int64(t)
	case uint64:
		return int64(t)
	case float32:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		if strings.TrimSpace(t) == "" {
			return 0
		}
		if n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
			return n
		}
		return 0
	default:
		return 0
	}
}

// GetAnyJSONFromCtx 尝试把 ctx 中的任意对象序列化为 JSON 字符串（用于日志字段），并做长度限制。
func GetAnyJSONFromCtx(c *gin.Context, key string, maxLen int) string {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	s := string(b)
	return TruncateString(s, maxLen)
}

func FmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		// json 数字默认 float64
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func GetClientIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	// Gin will respect trusted proxies settings if configured.
	ip := strings.TrimSpace(c.ClientIP())
	return ip
}
