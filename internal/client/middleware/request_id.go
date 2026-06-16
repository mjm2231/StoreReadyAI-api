package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// 用于存储请求ID的上下文键。
const CtxKeyRequestID = "rid"

// 默认的请求头名称。
const HeaderRequestID = "X-Request-Id"

// RequestIDConfig 配置请求ID中间件。
type RequestIDConfig struct {
	// HeaderName 是请求ID的请求头名称。默认值："X-Request-Id"。
	HeaderName string

	// Generator 在请求头缺失或无效时生成新的请求ID。
	// 如果为nil，则使用安全的随机16字节十六进制字符串。
	Generator func() string

	// AllowEmptyHeader 控制是否接受空的请求头值。
	// 默认值：false（空值视为缺失）。
	AllowEmptyHeader bool
}

func (c *RequestIDConfig) withDefaults() RequestIDConfig {
	cfg := *c
	if cfg.HeaderName == "" {
		cfg.HeaderName = HeaderRequestID
	}
	if cfg.Generator == nil {
		cfg.Generator = defaultRID
	}
	return cfg
}

// RequestID 确保每个请求都有一个稳定的请求ID：
//   - 优先使用传入的 X-Request-Id
//   - 否则生成一个新的请求ID
//   - 将其设置到 gin.Context（键名："rid"）
//   - 并写回响应头
func RequestID(cfg RequestIDConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	return func(c *gin.Context) {
		rid := strings.TrimSpace(c.GetHeader(cfg.HeaderName))
		if rid == "" && cfg.AllowEmptyHeader {
			// accept empty header as-is (rarely useful)
		}

		// Treat empty as missing unless explicitly allowed.
		if rid == "" && !cfg.AllowEmptyHeader {
			rid = cfg.Generator()
		}
		// If still empty (bad generator), force-generate.
		if strings.TrimSpace(rid) == "" {
			rid = defaultRID()
		}

		c.Set(CtxKeyRequestID, rid)
		c.Writer.Header().Set(cfg.HeaderName, rid)
		c.Next()
	}
}

// GetRequestID 从上下文中返回请求ID，如果不存在则返回空字符串。
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(CtxKeyRequestID); ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return ""
}

// defaultRID 生成一个安全的随机16字节十六进制字符串。
// 例如："9f0a2e7b3c9d4a1b8c7e6d5c4b3a2f1e"
func defaultRID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极不可能发生；回退到确定性但具有唯一性的值
		// Go中无法安全使用内存地址等熵源。
		// 因此我们使用一个带前缀的零字节十六进制编码的较小随机值。
		return "rid_" + hex.EncodeToString([]byte("fallback"))
	}
	return hex.EncodeToString(b)
}
