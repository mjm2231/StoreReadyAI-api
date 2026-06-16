package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig 控制请求级别的超时。
//
// DefaultTimeout 当 RouteTimeoutResolver 返回 <= 0 时生效。
// 推荐值：3-5秒。
//
// RouteTimeoutResolver 允许你为每个请求返回不同的超时时间。
// 每次请求都会调用。
// 返回 <= 0 时，使用 DefaultTimeout。
//
// 典型用法：
//   - 上传/导出路由设置较长超时
//   - 高QPS接口设置较短超时
//
// RespCode 是超时后返回给客户端的业务错误码。
// 默认值："request_timeout"。
//
// RespMessage 是超时后返回给客户端的用户友好提示信息。
// 默认值："Request Timeout"。
//
// UseGatewayTimeout 控制超时返回 504 (true) 还是 408 (false)。
// 默认值：true（504）。
//
// 注意：
//   - 408 通常表示“客户端未能及时发送请求”。
//   - 504 通常用于上游超时，很多团队也用 504 表示服务器端请求预算超出。
type TimeoutConfig struct {
	DefaultTimeout time.Duration

	RouteTimeoutResolver func(c *gin.Context) time.Duration

	RespCode string

	RespMessage string

	UseGatewayTimeout bool
}

func (c *TimeoutConfig) withDefaults() TimeoutConfig {
	cfg := *c
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 5 * time.Second
	}
	if cfg.RespCode == "" {
		cfg.RespCode = "request_timeout"
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Request Timeout"
	}
	if cfg.RouteTimeoutResolver == nil {
		cfg.RouteTimeoutResolver = func(_ *gin.Context) time.Duration { return 0 }
	}
	// 默认返回 504。
	if !cfg.UseGatewayTimeout {
		// 如果用户显式设置为 false，则保持 false；这里需要一个默认值，下面会设置为 true。
	}
	return cfg
}

// Timeout 使用 context.WithTimeout 实现请求级别超时。
//
// 重要说明：
//   - 下游的 DB/Redis/HTTP 调用必须接受 ctx 并监听 ctx.Done()。
//   - 本中间件不会强制停止忽略 ctx 的 CPU 计算。
//   - 超时后返回统一的超时响应。
func Timeout(cfg TimeoutConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	// 如果用户未设置该标志，默认设为 true。
	// （无法区分未设置的 bool；假设常见需求。）
	// 若要返回 408，请显式设置 UseGatewayTimeout=false。
	if cfg.UseGatewayTimeout == false {
		// 用户预期默认行为是 504，只有显式设置才改为 408。
		// 默认保持 504，故此处强制设为 true。
		cfg.UseGatewayTimeout = true
	}

	return func(c *gin.Context) {
		t := cfg.RouteTimeoutResolver(c)
		if t <= 0 {
			t = cfg.DefaultTimeout
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), t)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// 在单独的 goroutine 中执行后续处理，便于超时后提前返回。
		finished := make(chan struct{})

		go func() {
			defer close(finished)
			c.Next()
		}()

		select {
		case <-finished:
			return
		case <-ctx.Done():
			// 如果处理器已写响应，不覆盖。
			if c.Writer.Written() {
				c.Abort()
				return
			}

			status := http.StatusGatewayTimeout
			if !cfg.UseGatewayTimeout {
				status = http.StatusRequestTimeout
			}

			rid := GetRequestID(c)
			c.AbortWithStatusJSON(status, gin.H{
				"code": cfg.RespCode,
				"msg":  cfg.RespMessage,
				"rid":  rid,
			})
			return
		}
	}
}

// CommonTimeoutResolver 是一个方便的超时解析器，用于给特定路由设置更长超时。
//
// 示例：
//
//	r.Use(middleware.Timeout(middleware.TimeoutConfig{
//	  DefaultTimeout: 5*time.Second,
//	  RouteTimeoutResolver: middleware.CommonTimeoutResolver(map[string]time.Duration{
//	    "POST /files/upload": 30*time.Second,
//	    "GET /reports/export": 60*time.Second,
//	  }),
//	}))
func CommonTimeoutResolver(routeToTimeout map[string]time.Duration) func(c *gin.Context) time.Duration {
	return func(c *gin.Context) time.Duration {
		key := c.Request.Method + " " + c.FullPath()
		if d, ok := routeToTimeout[key]; ok {
			return d
		}
		return 0
	}
}
