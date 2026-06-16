package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig Tracing 中间件配置。
//
// 建议挂载顺序：
//   - 放在 RealIP / RequestID 之后
//   - 放在 AccessLog 之前（这样 AccessLog 可以拿到 trace_id/span_id）
//
// 说明：
//   - 本中间件使用全局 TracerProvider（由 internal/infra/tracer/otel.go 初始化）。
//   - 若未初始化 TracerProvider，本中间件也能工作，但不会真正导出数据。
//   - 会自动从请求头提取上游 trace context（W3C TraceContext），并创建 server span。
type TracingConfig struct {
	// Enabled 是否启用（默认 true）。
	Enabled bool

	// TracerName tracer 名称（默认："storeready_ai"）。
	TracerName string

	// SkipPathPrefixes 命中这些 path 前缀则不创建 span（默认：/health /metrics /favicon.ico）。
	SkipPathPrefixes []string

	// UseFullPath 是否优先使用 gin.FullPath 作为 route（默认 true）。
	UseFullPath bool

	// TraceIDKey 写入 gin.Context 的 key（默认："trace_id"）。
	TraceIDKey string
	// SpanIDKey 写入 gin.Context 的 key（默认："span_id"）。
	SpanIDKey string

	// RespTraceHeader 响应头写回 TraceID（默认："X-Trace-Id"）。
	RespTraceHeader string
	// RespSpanHeader 响应头写回 SpanID（默认："X-Span-Id"）。
	RespSpanHeader string

	// AddRouteAttribute 是否将 route 作为属性写入 span（默认 true）。
	AddRouteAttribute bool

	// AddUserContext 是否从 gin.Context 读取 uid/tenant_id 写入 span（默认 true）。
	AddUserContext bool
}

func (c *TracingConfig) withDefaults() TracingConfig {
	cfg := *c
	if !cfg.Enabled {
		// 显式关闭
		return cfg
	}
	// 默认启用
	cfg.Enabled = true
	if strings.TrimSpace(cfg.TracerName) == "" {
		cfg.TracerName = "storeready_ai"
	}
	if len(cfg.SkipPathPrefixes) == 0 {
		cfg.SkipPathPrefixes = []string{"/health", "/metrics", "/favicon.ico"}
	}
	// 默认使用 FullPath
	cfg.UseFullPath = true
	if strings.TrimSpace(cfg.TraceIDKey) == "" {
		cfg.TraceIDKey = "trace_id"
	}
	if strings.TrimSpace(cfg.SpanIDKey) == "" {
		cfg.SpanIDKey = "span_id"
	}
	if strings.TrimSpace(cfg.RespTraceHeader) == "" {
		cfg.RespTraceHeader = "X-Trace-Id"
	}
	if strings.TrimSpace(cfg.RespSpanHeader) == "" {
		cfg.RespSpanHeader = "X-Span-Id"
	}
	cfg.AddRouteAttribute = true
	cfg.AddUserContext = true
	return cfg
}

// Tracing Gin tracing 中间件：
//  1. 从请求头提取 trace context
//  2. 创建 server span
//  3. 将 trace_id/span_id 写入 gin.Context，便于 AccessLog/Audit 使用
//  4. 将 trace_id/span_id 写回响应头
func Tracing(cfg TracingConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		// 跳过基础端点
		if shouldSkipTracing(c, cfg.SkipPathPrefixes) {
			c.Next()
			return
		}

		// 解析 route/name
		method := ""
		path := ""
		if c.Request != nil {
			method = strings.ToUpper(strings.TrimSpace(c.Request.Method))
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
		}
		if method == "" {
			method = "UNKNOWN"
		}
		if path == "" {
			path = "unknown"
		}

		route := ""
		if cfg.UseFullPath {
			route = strings.TrimSpace(c.FullPath())
		}
		if route == "" {
			route = path
		}

		spanName := method + " " + route

		// 从请求头提取上游 context（W3C TraceContext/Baggage 等）
		ctx := c.Request.Context()
		prop := otel.GetTextMapPropagator()
		ctx = prop.Extract(ctx, propagation.HeaderCarrier(c.Request.Header))

		tr := otel.Tracer(cfg.TracerName)
		ctx, span := tr.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// 把 span context 写入 gin.Context（便于 AccessLog/Audit/安全事件）
		sc := span.SpanContext()
		if sc.IsValid() {
			traceID := sc.TraceID().String()
			spanID := sc.SpanID().String()
			c.Set(cfg.TraceIDKey, traceID)
			c.Set(cfg.SpanIDKey, spanID)
			// 写回响应头（便于前端/网关串联）
			c.Header(cfg.RespTraceHeader, traceID)
			c.Header(cfg.RespSpanHeader, spanID)
		}

		// 常用属性
		attrs := []attribute.KeyValue{
			attribute.String("http.method", method),
			attribute.String("http.path", path),
		}
		if cfg.AddRouteAttribute {
			attrs = append(attrs, attribute.String("http.route", route))
		}
		// 关联 RequestID（如果有）
		if rid := GetRequestID(c); rid != "" {
			attrs = append(attrs, attribute.String("rid", rid))
		}
		// 客户端 IP
		if ip := GetClientIP(c); ip != "" {
			attrs = append(attrs, attribute.String("client.ip", ip))
		}
		// 用户上下文
		if cfg.AddUserContext {
			if uid, ok := getStringFromCtxTracing(c, "uid"); ok {
				attrs = append(attrs, attribute.String("uid", uid))
			}
			if tid, ok := getStringFromCtxTracing(c, "tenant_id"); ok {
				attrs = append(attrs, attribute.String("tenant_id", tid))
			}
		}
		span.SetAttributes(attrs...)

		// 把 ctx 写回 request（让下游 handler/DB/HTTP client 使用同一个 trace 上下文）
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()

		// 结束时补充状态/耗时
		status := c.Writer.Status()
		costMS := time.Since(start).Milliseconds()
		span.SetAttributes(
			attribute.Int("http.status_code", status),
			attribute.Int64("http.cost_ms", costMS),
		)

		// 设置 span 状态
		if status >= 500 {
			span.SetStatus(codes.Error, "server_error")
			// 记录 gin.Errors 的简短摘要（避免泄露堆栈）
			if len(c.Errors) > 0 {
				msg := strings.TrimSpace(c.Errors.String())
				if msg == "" {
					msg = "gin_errors"
				}
				span.RecordError(errors.New(msg))
			}
		} else if status >= 400 {
			span.SetStatus(codes.Error, "client_error")
		} else {
			span.SetStatus(codes.Ok, "ok")
		}
	}
}

func shouldSkipTracing(c *gin.Context, prefixes []string) bool {
	p := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		p = c.Request.URL.Path
	}
	if p == "" {
		return false
	}
	for _, pre := range prefixes {
		if pre == "" {
			continue
		}
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// getStringFromCtxTracing 从 gin.Context 读取字符串（用于 tracing 属性）。
func getStringFromCtxTracing(c *gin.Context, key string) (string, bool) {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", false
		}
		return s, true
	}
	// 兜底：fmt
	ss := strings.TrimSpace(toStringTracing(v))
	if ss == "" {
		return "", false
	}
	return ss, true
}

func toStringTracing(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return "" // 仅用于属性，不强转数字，避免污染
	}
}
