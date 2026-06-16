package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"storeready_ai/internal/common"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditAction 审计动作（建议枚举化，但为了扩展性这里使用 string）。
type AuditAction string

// AuditRecord 企业级审计记录。
//
// 字段分组：
//   - who：谁做的
//   - what：做了什么
//   - where：从哪里做的
//   - result：结果如何
//   - meta：请求/响应摘要（脱敏/截断）
type AuditRecord struct {
	// 基础
	RID       string `json:"rid"`
	TraceID   string `json:"trace_id"`   // 预留：如后续接入 trace
	CreatedAt int64  `json:"created_at"` // unix 秒

	// who
	UID      string `json:"uid"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Scopes   string `json:"scopes"` // 逗号分隔，便于落库

	// what
	Action       string `json:"action"`        // 例如 user.create / auth.login / rbac.role.bind
	ResourceType string `json:"resource_type"` // 例如 user / role / pet
	ResourceID   string `json:"resource_id"`   // 资源 id（可空）

	// where
	IP     string `json:"ip"`
	UA     string `json:"ua"`
	Device string `json:"device"`
	Refer  string `json:"refer"`

	// result
	Success    bool   `json:"success"`
	HTTPStatus int    `json:"http_status"`
	ErrCode    string `json:"err_code"` // 业务错误码（尽量从响应中提取）
	LatencyMS  int64  `json:"latency_ms"`

	// meta
	Method         string `json:"method"`
	Path           string `json:"path"`
	QuerySummary   string `json:"query_summary"`   // 脱敏/截断
	BodySummary    string `json:"body_summary"`    // 脱敏/截断
	RespSummary    string `json:"resp_summary"`    // 可选：脱敏/截断
	RequestSizeB   int64  `json:"request_size_b"`  // 仅做参考
	ResponseSizeB  int64  `json:"response_size_b"` // 仅做参考
	ClientPlatform string `json:"client_platform"` // 预留

	// 风控（由 AntiBrush 等中间件注入，可选）
	RiskScore   int64  `json:"risk_score"`
	RiskAction  string `json:"risk_action"`
	RiskReasons string `json:"risk_reasons"`
}

// AuditWriter 落库写入器（建议同步写入，确保可追溯）。
type AuditWriter interface {
	Write(ctx context.Context, rec AuditRecord) error
}

// AuditQueuePublisher 可选：异步队列（用于流式分析/告警）。
// 失败不得影响主链路。
type AuditQueuePublisher interface {
	Publish(ctx context.Context, rec AuditRecord) error
}

// AuditWhatResolver 用于解析 what：action/resource。
//
// 返回：
//   - action：动作名（建议规范化）
//   - resourceType：资源类型
//   - resourceID：资源 id（可空）
//
// 如果返回空 action，则会用默认策略生成。
type AuditWhatResolver func(c *gin.Context) (action, resourceType, resourceID string)

// AuditConfig 审计中间件配置。
type AuditConfig struct {
	Writer AuditWriter
	Queue  AuditQueuePublisher

	// 是否记录响应摘要（默认 false）
	EnableRespSummary bool

	// 只记录关键写操作：默认仅 POST/PUT/PATCH/DELETE
	OnlyWriteMethods bool

	// OnlyWriteMethodsSet 表示 OnlyWriteMethods 是否被显式设置。
	// 用于解决 bool 无法区分“未设置(默认)”与“显式 false”的问题。
	OnlyWriteMethodsSet bool

	// 白名单：路径前缀命中直接不记录（health、metrics、swagger 等）
	SkipPathPrefixes []string

	// 可选：精确匹配 gin.FullPath（命中不记录）
	SkipFullPaths map[string]struct{}

	// what 解析器
	WhatResolver AuditWhatResolver

	// 请求体读取上限（默认 8KB），超过则不读 body
	MaxBodyBytes int64

	// 摘要字段最大长度（默认 2048 字符）
	MaxSummaryLen int

	// 脱敏字段（默认包含 password/token/secret/code 等）
	MaskKeys []string

	// 是否在 gin.Context 设置 "audit_record"（便于 handler/业务补充 meta）
	InjectToContext bool
}

func (c *AuditConfig) withDefaults() AuditConfig {
	cfg := *c
	// 默认仅记录写操作；如果调用方显式设置，则尊重调用方
	if !cfg.OnlyWriteMethodsSet {
		cfg.OnlyWriteMethods = true
	}
	if cfg.SkipFullPaths == nil {
		cfg.SkipFullPaths = map[string]struct{}{}
	}
	if cfg.WhatResolver == nil {
		cfg.WhatResolver = defaultAuditWhatResolver
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 8 * 1024
	}
	if cfg.MaxSummaryLen <= 0 {
		cfg.MaxSummaryLen = 2048
	}
	if len(cfg.MaskKeys) == 0 {
		cfg.MaskKeys = []string{
			"password", "passwd", "pwd",
			"token", "access_token", "refresh_token", "id_token",
			"secret", "client_secret", "api_key", "apikey",
			"authorization",
			"code", "sms_code", "otp",
		}
	}
	if len(cfg.SkipPathPrefixes) == 0 {
		cfg.SkipPathPrefixes = []string{"/health", "/metrics", "/swagger", "/favicon.ico"}
	}
	return cfg
}

// Audit 关键操作审计中间件。
func Audit(cfg AuditConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	if cfg.Writer == nil {
		panic("Audit: Writer 不能为空")
	}

	return func(c *gin.Context) {
		// 0) 过滤：白名单不记录
		if shouldSkipAudit(c, cfg) {
			c.Next()
			return
		}
		// 1) 过滤：只记录写操作
		if cfg.OnlyWriteMethods {
			m := strings.ToUpper(strings.TrimSpace(c.Request.Method))
			switch m {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				// ok
			default:
				c.Next()
				return
			}
		}

		start := time.Now()

		// 2) 捕获响应（状态码 + 可选摘要）
		cw := &auditCaptureWriter{ResponseWriter: c.Writer, maxBytes: cfg.MaxBodyBytes}
		c.Writer = cw

		// 3) 提前收集请求摘要（轻量 + 脱敏）
		querySum := summarizeQuery(c.Request.URL, cfg.MaxSummaryLen, cfg.MaskKeys)
		bodySum, reqSize := summarizeBody(c, cfg.MaxBodyBytes, cfg.MaxSummaryLen, cfg.MaskKeys)

		// 4) 执行业务
		c.Next()

		// 5) 组装 record
		lat := time.Since(start)
		status := cw.status
		if status == 0 {
			status = http.StatusOK
		}
		success := status < 400

		action, rtype, rid := cfg.WhatResolver(c)
		if strings.TrimSpace(action) == "" {
			action = defaultAuditAction(c)
		}

		uid := common.GetStringFromCtx(c, "uid")
		tenant := common.GetStringFromCtx(c, "tenant_id")
		role := common.GetStringFromCtx(c, "role")
		scopes := getScopesFromCtx(c)

		ip := common.GetClientIP(c)
		ua := strings.TrimSpace(c.GetHeader("User-Agent"))
		device := strings.TrimSpace(common.FirstNonEmpty(
			c.GetHeader("X-Device-Id"),
			c.GetHeader("X-Device-ID"),
			c.GetHeader("X-Client-Id"),
			c.GetHeader("X-Client-ID"),
		))
		refer := strings.TrimSpace(c.GetHeader("Referer"))

		errCode := extractErrCodeFromJSON(cw.body)
		respSum := ""
		if cfg.EnableRespSummary {
			respSum = summarizeBytesAsJSON(cw.body, cfg.MaxSummaryLen, cfg.MaskKeys)
		}

		rec := AuditRecord{
			RID:       GetRequestID(c),
			TraceID:   common.FirstNonEmpty(common.GetStringFromCtx(c, "trace_id"), strings.TrimSpace(c.GetHeader("X-Trace-Id"))),
			CreatedAt: start.Unix(),

			UID:      uid,
			TenantID: tenant,
			Role:     role,
			Scopes:   scopes,

			Action:       action,
			ResourceType: rtype,
			ResourceID:   rid,

			IP:     ip,
			UA:     ua,
			Device: device,
			Refer:  refer,

			Success:    success,
			HTTPStatus: status,
			ErrCode:    errCode,
			LatencyMS:  lat.Milliseconds(),

			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			QuerySummary:  querySum,
			BodySummary:   bodySum,
			RespSummary:   respSum,
			RequestSizeB:  reqSize,
			ResponseSizeB: int64(len(cw.body)),

			RiskScore:   common.GetInt64FromCtx(c, "risk_score"),
			RiskAction:  getStringFromCtxValue(c, "risk_action"),
			RiskReasons: common.GetAnyJSONFromCtx(c, "risk_reasons", cfg.MaxSummaryLen),
		}

		if cfg.InjectToContext {
			c.Set("audit_record", rec)
		}

		// 6) 写入：落库为主（建议同步写）
		_ = cfg.Writer.Write(c.Request.Context(), rec)

		// 7) 可选：异步队列（失败不影响主链路）
		if cfg.Queue != nil {
			_ = cfg.Queue.Publish(c.Request.Context(), rec)
		}
	}
}

func shouldSkipAudit(c *gin.Context, cfg AuditConfig) bool {
	fp := c.FullPath()
	if fp != "" {
		if _, ok := cfg.SkipFullPaths[fp]; ok {
			return true
		}
	}
	p := ""
	if c.Request != nil && c.Request.URL != nil {
		p = c.Request.URL.Path
	}
	for _, pre := range cfg.SkipPathPrefixes {
		if pre == "" {
			continue
		}
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

func defaultAuditWhatResolver(c *gin.Context) (action, resourceType, resourceID string) {
	// 你也可以在 handler 中主动设置：
	//   c.Set("audit_action", "rbac.role.bind")
	//   c.Set("audit_resource_type", "role")
	//   c.Set("audit_resource_id", roleID)

	action = strings.TrimSpace(common.GetStringFromCtx(c, "audit_action"))

	resourceType = strings.TrimSpace(common.GetStringFromCtx(c, "audit_resource_type"))

	resourceID = strings.TrimSpace(common.GetStringFromCtx(c, "audit_resource_id"))
	return
}

func defaultAuditAction(c *gin.Context) string {
	// 默认：method + path（规范化）
	m := strings.ToLower(strings.TrimSpace(c.Request.Method))
	p := strings.TrimSpace(c.FullPath())
	if p == "" {
		p = strings.TrimSpace(c.Request.URL.Path)
	}
	p = strings.Trim(p, "/")
	p = strings.ReplaceAll(p, "/", ".")
	if p == "" {
		p = "root"
	}
	return m + "." + p
}

// auditCaptureWriter 捕获响应状态与 body（限制大小）。
type auditCaptureWriter struct {
	gin.ResponseWriter
	status   int
	body     []byte
	maxBytes int64
}

func (w *auditCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditCaptureWriter) Write(b []byte) (int, error) {
	// 只缓存有限字节，避免占用过多内存
	if w.maxBytes > 0 {
		remain := int(w.maxBytes) - len(w.body)
		if remain > 0 {
			if len(b) <= remain {
				w.body = append(w.body, b...)
			} else {
				w.body = append(w.body, b[:remain]...)
			}
		}
	}
	return w.ResponseWriter.Write(b)
}

// summarizeQuery 对 query 做脱敏摘要。
func summarizeQuery(u *url.URL, maxLen int, maskKeys []string) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	if len(q) == 0 {
		return ""
	}
	m := map[string]any{}
	for k, vs := range q {
		if len(vs) == 1 {
			m[k] = vs[0]
		} else {
			m[k] = vs
		}
	}
	return summarizeMapAsJSON(m, maxLen, maskKeys)
}

// summarizeBody 读取小 body 并恢复 body（仅用于摘要，不影响业务读取）。
func summarizeBody(c *gin.Context, maxBodyBytes int64, maxLen int, maskKeys []string) (summary string, size int64) {
	if c == nil || c.Request == nil {
		return "", 0
	}
	// ContentLength 仅参考
	size = c.Request.ContentLength

	// 只在 body 很小/或未知但我们限制读取时尝试
	if maxBodyBytes <= 0 {
		return "", size
	}
	if c.Request.Body == nil {
		return "", size
	}
	if c.Request.ContentLength > maxBodyBytes {
		return "", size
	}

	ct := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
	// 仅对 json/x-www-form-urlencoded/multipart(不读) 做处理
	if strings.HasPrefix(ct, "multipart/") {
		return "", size
	}

	b, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyBytes))
	if err != nil {
		return "", size
	}
	// 恢复 body，保证下游还能读取
	c.Request.Body = io.NopCloser(bytes.NewReader(b))
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return "", size
	}

	// json
	if strings.HasPrefix(ct, "application/json") {
		return summarizeBytesAsJSON(b, maxLen, maskKeys), size
	}
	// form
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		vals, err := url.ParseQuery(string(b))
		if err != nil {
			return common.TruncateString(string(b), maxLen), size
		}
		m := map[string]any{}
		for k, vs := range vals {
			if len(vs) == 1 {
				m[k] = vs[0]
			} else {
				m[k] = vs
			}
		}
		return summarizeMapAsJSON(m, maxLen, maskKeys), size
	}

	// 其他：仅截断原文
	return common.TruncateString(string(b), maxLen), size
}

func summarizeBytesAsJSON(b []byte, maxLen int, maskKeys []string) string {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return common.TruncateString(string(b), maxLen)
	}
	masked := maskJSON(v, maskKeys)
	out, err := json.Marshal(masked)
	if err != nil {
		return common.TruncateString(string(b), maxLen)
	}
	return common.TruncateString(string(out), maxLen)
}

func summarizeMapAsJSON(m map[string]any, maxLen int, maskKeys []string) string {
	masked := maskJSON(m, maskKeys)
	out, err := json.Marshal(masked)
	if err != nil {
		return ""
	}
	return common.TruncateString(string(out), maxLen)
}

// maskJSON 对常见敏感字段进行脱敏（递归）。
func maskJSON(v any, maskKeys []string) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			if isMaskedKey(k, maskKeys) {
				out[k] = "***"
				continue
			}
			out[k] = maskJSON(vv, maskKeys)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, it := range t {
			out = append(out, maskJSON(it, maskKeys))
		}
		return out
	default:
		return v
	}
}

func isMaskedKey(k string, maskKeys []string) bool {
	kk := strings.ToLower(strings.TrimSpace(k))
	if kk == "" {
		return false
	}
	for _, mk := range maskKeys {
		mkk := strings.ToLower(strings.TrimSpace(mk))
		if mkk == "" {
			continue
		}
		if kk == mkk {
			return true
		}
		// 兼容常见命名：xxx_password / password_xxx
		if strings.Contains(kk, mkk) {
			return true
		}
	}
	return false
}

// extractErrCodeFromJSON 尝试从响应 JSON 中提取业务 code 字段。
//
// 兼容常见格式：
//
//	{"code":0,"msg":"OK"}
//	{"code":"xxx","msg":""}
//	{"err_code":"xxx"}
func extractErrCodeFromJSON(resp []byte) string {
	resp = bytes.TrimSpace(resp)
	if len(resp) == 0 {
		return ""
	}
	if len(resp) > 32*1024 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(resp, &m); err != nil {
		return ""
	}
	if v, ok := m["err_code"]; ok {
		return fmt.Sprint(v)
	}
	if v, ok := m["code"]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func getScopesFromCtx(c *gin.Context) string {
	v, ok := c.Get("scopes")
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case []string:
		return strings.Join(t, ",")
	case string:
		return strings.TrimSpace(t)
	default:
		return fmt.Sprint(t)
	}
}

// getStringFromCtxValue 从 gin.Context 读取字符串（如果不存在返回空）。
func getStringFromCtxValue(c *gin.Context, key string) string {
	return strings.TrimSpace(common.GetStringFromCtx(c, key))
}

// 这些错误仅用于内部提示，避免对外暴露。
var (
	ErrAuditWriteFailed = errors.New("audit write failed")
)
