package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"storeready_ai/internal/security"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryConfig 控制 panic 恢复行为。
type RecoveryConfig struct {
	// Logger 用于写结构化的 panic 日志。
	// 如果为空，则使用 zap.L() 作为默认 Logger。
	Logger *zap.Logger

	// RespCode 返回给客户端的业务错误码。
	// 默认值: "internal_error"。
	RespCode string

	// RespMessage 返回给客户端的用户友好消息。
	// 默认值: "Internal Server Error"。
	RespMessage string

	// IncludeRequestBody 控制是否记录请求体。
	// 默认值: false。请注意避免泄露敏感信息。
	IncludeRequestBody bool

	// MaxRequestBodyBytes 限制记录请求体的最大字节数。
	// 默认值: 2048。
	MaxRequestBodyBytes int

	// SecurityEmitter 安全事件发射器（可选）。
	// 用于把 panic 恢复沉淀为安全事件，便于后续告警/BI。
	// 注意：仅当 emitter 非空且 Writer/Queue 至少一个存在时才会写事件。
	SecurityEmitter *security.SecurityEventEmitter
}

func (c *RecoveryConfig) withDefaults() RecoveryConfig {
	cfg := *c
	if cfg.Logger == nil {
		cfg.Logger = zap.L()
	}
	if cfg.RespCode == "" {
		cfg.RespCode = "internal_error"
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Internal Server Error"
	}
	if cfg.MaxRequestBodyBytes <= 0 {
		cfg.MaxRequestBodyBytes = 2048
	}
	return cfg
}

// Recovery 捕获 panic，结构化记录堆栈信息，并返回统一错误响应。
//
// 同时会发出类似 "security/audit" 的日志，因为 panic 是严重异常。
func Recovery(cfg RecoveryConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				rid := GetRequestID(c)
				ip := GetClientIP(c)

				stack := trimStack(debug.Stack(), 64*1024)

				// 统一沉淀安全事件（可选）：不影响主链路
				if cfg.SecurityEmitter != nil && (cfg.SecurityEmitter.Writer != nil || cfg.SecurityEmitter.Queue != nil) {
					details := map[string]any{
						"reason":    "panic",
						"panic":     fmt.Sprint(rec),
						"path":      c.Request.URL.Path,
						"method":    c.Request.Method,
						"ua":        c.Request.UserAgent(),
						"query":     c.Request.URL.RawQuery,
						"referer":   c.Request.Referer(),
						"stack_len": len(stack),
					}
					if cfg.IncludeRequestBody {
						if v, ok := c.Get("req_body"); ok {
							if b, ok2 := v.([]byte); ok2 {
								details["req_body_len"] = len(b)
							}
						}
					}
					_ = security.EmitSecurityEvent(c, cfg.SecurityEmitter, "panic_recovered", security.SecuritySeverityCritical, details)
				}

				fields := []zap.Field{
					zap.String("event", "panic"),
					zap.Any("panic", rec),
					zap.String("rid", rid),
					zap.String("ip", ip),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("ua", c.Request.UserAgent()),
					zap.String("referer", c.Request.Referer()),
					zap.ByteString("stack", stack),
				}

				// 如果请求体需要记录（仅能捕获已存储于 ctx 的请求体）
				if cfg.IncludeRequestBody {
					if v, ok := c.Get("req_body"); ok {
						if b, ok2 := v.([]byte); ok2 && len(b) > 0 {
							fields = append(fields, zap.ByteString("req_body", truncateBytes(b, cfg.MaxRequestBodyBytes)))
						}
					}
				}

				// 避免重复写响应
				if c.Writer.Written() {
					c.Abort()
					return
				}

				// 严重错误日志
				cfg.Logger.Error("security.panic", fields...)

				// 统一响应，不泄露内部细节
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": cfg.RespCode,
					"msg":  cfg.RespMessage,
					"rid":  rid,
				})
			}
		}()

		c.Next()
	}
}

func truncateBytes(b []byte, max int) []byte {
	if max <= 0 || len(b) <= max {
		return b
	}
	return b[:max]
}

func trimStack(stack []byte, max int) []byte {
	if max <= 0 || len(stack) <= max {
		return stack
	}
	// 保留尾部，通常包含最相关的堆栈帧
	if max < 256 {
		max = 256
	}
	return stack[len(stack)-max:]
}

// FormatRecoveredPanic 是一个辅助函数，可在其他地方（测试/指标）复用。
func FormatRecoveredPanic(rec any) string {
	var buf bytes.Buffer
	buf.WriteString("panic recovered: ")
	buf.WriteString(fmt.Sprint(rec))
	return buf.String()
}
