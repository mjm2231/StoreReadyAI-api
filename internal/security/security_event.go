package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"storeready_ai/internal/common"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecuritySeverity 安全事件严重级别。
//
// 建议约定：
//   - info：一般安全相关事件（例如被动记录）
//   - low：低风险（轻度限流、轻度规则命中）
//   - medium：中风险（多次命中、需要验证码/风控升级）
//   - high：高风险（重度封禁、疑似攻击）
//   - critical：严重（panic、越权、批量扫描、疑似入侵）
type SecuritySeverity string

const (
	SecuritySeverityInfo     SecuritySeverity = "info"
	SecuritySeverityLow      SecuritySeverity = "low"
	SecuritySeverityMedium   SecuritySeverity = "medium"
	SecuritySeverityHigh     SecuritySeverity = "high"
	SecuritySeverityCritical SecuritySeverity = "critical"
)

// SecurityEvent 安全事件。
//
// 目的：把 Firewall/RateLimit/AntiBrush/Recovery 等中间件的命中与异常行为统一沉淀下来，
// 后续可用于：封禁、告警、BI 分析。
//
// 字段尽量保持稳定，便于落库与分析。
type SecurityEvent struct {
	// 基础
	Type      string           `json:"type"`
	Severity  SecuritySeverity `json:"severity"`
	RID       string           `json:"rid"`
	CreatedAt int64            `json:"created_at"` // unix 秒

	// who
	UID      string `json:"uid"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`

	// where
	IP     string `json:"ip"`
	UA     string `json:"ua"`
	Device string `json:"device"`

	// what
	Route  string `json:"route"`
	Method string `json:"method"`

	// details（统一 JSON 字符串，避免大对象/结构漂移）
	Details string `json:"details"`
}

// SecurityEventWriter 落库写入器（建议同步写入，保证可追溯）。
type SecurityEventWriter interface {
	Write(ctx context.Context, ev SecurityEvent) error
}

// SecurityEventQueuePublisher 可选：异步队列发布器（用于流式分析/告警）。
// 失败不得影响主链路。
type SecurityEventQueuePublisher interface {
	Publish(ctx context.Context, ev SecurityEvent) error
}

// SecurityEventEmitter 事件发射器。
//
// 约定：
//   - Writer 建议必填（落库为主）
//   - Queue 可选（失败不影响主链路）
//   - 任何错误都返回给调用方，但调用方通常可以忽略
//
// 注意：
//   - 不建议在这里做重 CPU 的序列化/正则等
//   - 深度 WAF/分析交给网关/专用风控系统
type SecurityEventEmitter struct {
	Writer SecurityEventWriter
	Queue  SecurityEventQueuePublisher

	// details 最大长度（默认 2048）
	MaxDetailsLen int
}

func (e *SecurityEventEmitter) withDefaults() *SecurityEventEmitter {
	if e == nil {
		return &SecurityEventEmitter{MaxDetailsLen: 2048}
	}
	if e.MaxDetailsLen <= 0 {
		e.MaxDetailsLen = 2048
	}
	return e
}

// Emit 发送安全事件：同步落库 + 可选异步队列。
func (e *SecurityEventEmitter) Emit(ctx context.Context, ev SecurityEvent) error {
	e = e.withDefaults()
	// 不强制要求 Writer，但企业级建议落库为主
	var firstErr error
	if e.Writer != nil {
		if err := e.Writer.Write(ctx, ev); err != nil {
			firstErr = err
		}
	}
	if e.Queue != nil {
		if err := e.Queue.Publish(ctx, ev); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// EmitSecurityEvent 中间件便捷方法：自动补齐 rid/ip/uid/tenant/route/method 等字段。
//
// 使用示例：
//
//	_ = security.EmitSecurityEvent(c, emitter, "rate_limit_hit", security.SecuritySeverityLow, map[string]any{"key": key})
func EmitSecurityEvent(c *gin.Context, emitter *SecurityEventEmitter, typ string, severity SecuritySeverity, details map[string]any) error {
	ev := SecurityEvent{
		Type:     strings.TrimSpace(typ),
		Severity: severity,
	}
	fillSecurityEventFromContext(c, &ev)
	ev.Details = encodeDetails(details, emitter)
	return EmitSecurityEventWithFields(c, emitter, ev)
}

// EmitSecurityEventWithFields 允许调用方自定义更多字段（例如覆写 route/resource）。
//
// 仍然会补齐缺省字段（RID/IP/UID/Tenant/Route/Method/UA/Device/CreatedAt）。
func EmitSecurityEventWithFields(c *gin.Context, emitter *SecurityEventEmitter, ev SecurityEvent) error {
	emitter = emitter.withDefaults()
	// 空 emitter：直接忽略
	if emitter.Writer == nil && emitter.Queue == nil {
		return nil
	}
	// 补齐字段
	if strings.TrimSpace(ev.Type) == "" {
		ev.Type = "security_event"
	}
	if strings.TrimSpace(string(ev.Severity)) == "" {
		ev.Severity = SecuritySeverityInfo
	}
	fillMissingSecurityEventFromContext(c, &ev)
	// details 兜底截断
	if emitter.MaxDetailsLen > 0 && len(ev.Details) > emitter.MaxDetailsLen {
		ev.Details = ev.Details[:emitter.MaxDetailsLen] + "...(truncated)"
	}
	return emitter.Emit(c.Request.Context(), ev)
}

func fillSecurityEventFromContext(c *gin.Context, ev *SecurityEvent) {
	if ev == nil || c == nil {
		return
	}
	// 创建时间
	ev.CreatedAt = time.Now().Unix()

	ev.RID = getRequestID(c)
	ev.IP = common.GetClientIP(c)

	if uid, ok := getStringFromCtxLite(c, "uid"); ok {
		ev.UID = uid
	}
	if tid, ok := getStringFromCtxLite(c, "tenant_id"); ok {
		ev.TenantID = tid
	}
	if role, ok := getStringFromCtxLite(c, "role"); ok {
		ev.Role = role
	}

	ev.UA = strings.TrimSpace(c.GetHeader("User-Agent"))
	ev.Device = strings.TrimSpace(firstNonEmpty(
		c.GetHeader("X-Device-Id"),
		c.GetHeader("X-Device-ID"),
		c.GetHeader("X-Client-Id"),
		c.GetHeader("X-Client-ID"),
	))

	// 路由
	route := strings.TrimSpace(c.FullPath())
	if route == "" && c.Request != nil && c.Request.URL != nil {
		route = strings.TrimSpace(c.Request.URL.Path)
	}
	ev.Route = route
	if c.Request != nil {
		ev.Method = strings.TrimSpace(c.Request.Method)
	}
}

// fillMissingSecurityEventFromContext 仅在字段为空时补齐，避免覆盖调用方主动设置。
func fillMissingSecurityEventFromContext(c *gin.Context, ev *SecurityEvent) {
	if ev == nil || c == nil {
		return
	}
	if ev.CreatedAt <= 0 {
		ev.CreatedAt = time.Now().Unix()
	}
	if strings.TrimSpace(ev.RID) == "" {
		ev.RID = getRequestID(c)
	}
	if strings.TrimSpace(ev.IP) == "" {
		ev.IP = common.GetClientIP(c)
	}
	if strings.TrimSpace(ev.UID) == "" {
		if uid, ok := getStringFromCtxLite(c, "uid"); ok {
			ev.UID = uid
		}
	}
	if strings.TrimSpace(ev.TenantID) == "" {
		if tid, ok := getStringFromCtxLite(c, "tenant_id"); ok {
			ev.TenantID = tid
		}
	}
	if strings.TrimSpace(ev.Role) == "" {
		if role, ok := getStringFromCtxLite(c, "role"); ok {
			ev.Role = role
		}
	}
	if strings.TrimSpace(ev.UA) == "" {
		ev.UA = strings.TrimSpace(c.GetHeader("User-Agent"))
	}
	if strings.TrimSpace(ev.Device) == "" {
		ev.Device = strings.TrimSpace(firstNonEmpty(
			c.GetHeader("X-Device-Id"),
			c.GetHeader("X-Device-ID"),
			c.GetHeader("X-Client-Id"),
			c.GetHeader("X-Client-ID"),
		))
	}
	if strings.TrimSpace(ev.Route) == "" {
		route := strings.TrimSpace(c.FullPath())
		if route == "" && c.Request != nil && c.Request.URL != nil {
			route = strings.TrimSpace(c.Request.URL.Path)
		}
		ev.Route = route
	}
	if strings.TrimSpace(ev.Method) == "" {
		if c.Request != nil {
			ev.Method = strings.TrimSpace(c.Request.Method)
		}
	}
}

func encodeDetails(details map[string]any, emitter *SecurityEventEmitter) string {
	if details == nil {
		return ""
	}
	b, err := json.Marshal(details)
	if err != nil {
		// 兜底：fmt
		return fmt.Sprint(details)
	}
	out := string(b)
	// 截断
	if emitter != nil {
		emitter = emitter.withDefaults()
		if emitter.MaxDetailsLen > 0 && len(out) > emitter.MaxDetailsLen {
			out = out[:emitter.MaxDetailsLen] + "...(truncated)"
		}
	}
	return out
}

func getRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	// Prefer values set by upstream middleware.
	if rid := strings.TrimSpace(c.GetString("rid")); rid != "" {
		return rid
	}
	// Common header names.
	for _, k := range []string{"X-Request-Id", "X-Request-ID", "X-RequestID", "X-Correlation-Id", "X-Correlation-ID"} {
		if v := strings.TrimSpace(c.GetHeader(k)); v != "" {
			return v
		}
	}
	return ""
}

func getStringFromCtxLite(c *gin.Context, key string) (string, bool) {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return "", false
		}
		return t, true
	case int:
		return strconv.FormatInt(int64(t), 10), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case uint:
		return strconv.FormatUint(uint64(t), 10), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	default:
		return fmt.Sprint(t), true
	}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ErrSecurityEventEmitFailed 仅用于内部标识错误，不建议对外透传。
var ErrSecurityEventEmitFailed = errors.New("security event emit failed")
