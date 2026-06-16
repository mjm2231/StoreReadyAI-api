package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"storeready_ai/internal/common"
	"storeready_ai/internal/infra/hander"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AccessLogConfig 访问日志中间件配置。
//
// 建议：放在中间件链路最后，确保拿到最终的 status、cost、uid 等信息。
type AccessLogConfig struct {
	// Logger 结构化日志输出（可选）。为空则使用 zap.L()。
	Logger *zap.Logger

	// SkipPathPrefixes 路径前缀命中则不记录（可选）。
	// 默认：/health /metrics /favicon.ico
	SkipPathPrefixes []string

	// MaxRespBodyBytes 缓存响应体用于提取 err_code 的最大字节数。
	// 默认 32KB。
	MaxRespBodyBytes int
}

func (c *AccessLogConfig) withDefaults() AccessLogConfig {
	cfg := *c
	if cfg.Logger == nil {
		cfg.Logger = zap.L()
	}
	if len(cfg.SkipPathPrefixes) == 0 {
		cfg.SkipPathPrefixes = []string{"/health", "/metrics", "/favicon.ico"}
	}
	if cfg.MaxRespBodyBytes <= 0 {
		cfg.MaxRespBodyBytes = 32 * 1024
	}
	return cfg
}

// AccessLog 结构化访问日志中间件。
func AccessLog(cfg AccessLogConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()

	return func(c *gin.Context) {

		// 过滤：跳过健康检查/metrics 等
		if shouldSkipAccessLog(c, cfg.SkipPathPrefixes) {
			c.Next()
			return
		}

		start := time.Now()

		// 包装 writer：捕获 status、bytes_out、并缓存有限响应体用于提取业务 err_code
		cw := &accessLogCaptureWriter{ResponseWriter: c.Writer, status: 200, maxBody: cfg.MaxRespBodyBytes}
		c.Writer = cw

		// 执行业务
		c.Next()

		// 构建字段
		costMS := time.Since(start).Milliseconds()
		status := cw.status
		if status == 0 {
			status = c.Writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
		}

		method := ""
		path := ""
		referer := ""
		ua := ""
		bytesIn := int64(0)
		if c.Request != nil {
			method = strings.ToUpper(strings.TrimSpace(c.Request.Method))
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
			referer = strings.TrimSpace(c.Request.Referer())
			ua = strings.TrimSpace(c.Request.UserAgent())
			if c.Request.ContentLength > 0 {
				bytesIn = c.Request.ContentLength
			}
		}
		if method == "" {
			method = "UNKNOWN"
		}
		if path == "" {
			path = "unknown"
		}

		route := strings.TrimSpace(c.FullPath())
		if route == "" {
			route = path
		}

		ip := GetClientIP(c)
		rid := GetRequestID(c)

		traceID := common.FirstNonEmpty(common.GetStringFromCtx(c, common.CtxKeyTraceID), strings.TrimSpace(c.GetHeader("X-Trace-Id")))
		spanID := common.FirstNonEmpty(common.GetStringFromCtx(c, common.CtxKeySpanID), strings.TrimSpace(c.GetHeader("X-Span-Id")))

		uid := common.GetStringFromCtx(c, common.CtxKeyUID)
		tenantID := common.GetStringFromCtx(c, hander.CtxKeyTenantID)
		errCode := extractErrCodeFromResp(cw.body)

		bytesOut := cw.bytesOut

		// 可选：风控信息（由 AntiBrush 等中间件注入）
		riskScore := common.GetInt64FromCtx(c, "risk_score")
		riskAction := common.GetStringFromCtx(c, "risk_action")
		riskReasons := common.GetAnyJSONFromCtx(c, "risk_reasons", 2048)

		level := accessLogLevel(status)
		fields := []zap.Field{
			zap.String("rid", rid),
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),

			zap.String("method", method),
			zap.String("path", path),
			zap.String("route", route),
			zap.Int("status", status),
			zap.Int64("cost_ms", costMS),

			zap.String("ip", ip),
			zap.String("ua", ua),
			zap.String("referer", referer),

			zap.String("uid", uid),
			zap.String("tenant_id", tenantID),

			zap.String("err_code", errCode),
			zap.Int64("bytes_in", bytesIn),
			zap.Int64("bytes_out", bytesOut),

			zap.Int64("risk_score", riskScore),
			zap.String("risk_action", riskAction),
			zap.String("risk_reasons", riskReasons),
		}

		// 如果 gin.Errors 有内容，也追加一个简短摘要（避免泄露内部堆栈）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("gin_errors", common.TruncateString(c.Errors.String(), 1024)))
		}

		msg := "http.access"
		switch level {
		case zapcore.ErrorLevel:
			cfg.Logger.Error(msg, fields...)
		case zapcore.WarnLevel:
			cfg.Logger.Warn(msg, fields...)
		default:
			cfg.Logger.Info(msg, fields...)
		}
	}
}

func shouldSkipAccessLog(c *gin.Context, prefixes []string) bool {
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

func accessLogLevel(status int) zapcore.Level {
	if status >= 500 {
		return zapcore.ErrorLevel
	}
	if status >= 400 {
		return zapcore.WarnLevel
	}
	return zapcore.InfoLevel
}

// accessLogCaptureWriter 捕获响应 status 与 bytes_out，并缓存少量响应体用于提取业务 err_code。
type accessLogCaptureWriter struct {
	gin.ResponseWriter
	status   int
	bytesOut int64
	body     []byte
	maxBody  int
}

func (w *accessLogCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *accessLogCaptureWriter) Write(b []byte) (int, error) {
	// 统计输出字节
	w.bytesOut += int64(len(b))

	// 缓存有限响应体
	if w.maxBody > 0 && len(w.body) < w.maxBody {
		remain := w.maxBody - len(w.body)
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

// extractErrCodeFromResp 尝试从响应 JSON 中提取业务错误码。
//
// 兼容格式：
//
//	{"code":0,"msg":"OK"}
//	{"code":"xxx","msg":""}
//	{"err_code":"xxx"}
func extractErrCodeFromResp(resp []byte) string {
	resp = bytes.TrimSpace(resp)
	if len(resp) == 0 {
		return ""
	}
	// 只在看起来像 json 的情况下解析
	if resp[0] != '{' && resp[0] != '[' {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(resp, &m); err != nil {
		return ""
	}
	if v, ok := m["err_code"]; ok {
		return common.FmtAny(v)
	}
	if v, ok := m["code"]; ok {
		return common.FmtAny(v)
	}
	return ""
}
