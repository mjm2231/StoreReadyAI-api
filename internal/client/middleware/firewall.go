package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"storeready_ai/internal/security"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// FirewallConfig 轻量 WAF 配置。
//
// 设计目标：
//   - 轻：只做低成本的快速过滤
//   - 默认可用：不需要复杂依赖
//   - 安全：对明显恶意流量尽早拒绝，节省后端资源
//
// 注意：
//   - 深度 WAF（复杂规则、正则大扫、语义检测）建议交给网关/Nginx/云 WAF。
//   - 本中间件主要为“挡明显攻击/扫描/异常请求”提供第一道闸。
type FirewallConfig struct {
	// 允许的 HTTP 方法白名单。
	// 默认：GET,POST,PUT,PATCH,DELETE
	AllowMethods []string

	// 普通 API 最大 body 大小（字节）。
	// 默认：2MB
	MaxBodyBytes int64

	// 上传/导出等特殊路由最大 body 大小（字节）。
	// 默认：20MB
	MaxBodyBytesLarge int64

	// LargePathPrefixes 命中这些 path 前缀的请求，使用 MaxBodyBytesLarge。
	// 例如：/files/upload, /reports/export
	LargePathPrefixes []string

	// IP 黑名单：命中直接拒绝（403）。
	IPBlockList []string

	// IP 灰名单：命中返回 429（可理解为临时限制/降级）。
	IPGrayList []string

	// User-Agent 为空是否直接拒绝。
	// 默认：false（仅降权：这里实现为“不拒绝”，交给 RateLimit/风控处理）
	RejectEmptyUA bool

	// 直接拒绝的扫描器 UA 关键字（小写匹配）。
	// 默认包含常见扫描器关键字。
	BlockUAKeywords []string

	// Referer 为空是否直接拒绝。
	// 默认：false（很多 API 正常不带 referer）
	RejectEmptyReferer bool

	// Content-Type 校验：当路径匹配 JsonPathPrefixes 时，要求 Content-Type 为 application/json（允许带 charset）。
	JsonPathPrefixes []string

	// 禁止的路径特征：包含这些字符串直接拒绝。
	// 默认：..  //  \
	BlockPathSubstrings []string

	// ForbiddenCode 命中拒绝规则时的业务错误码。
	// 默认："firewall_forbidden"
	ForbiddenCode string

	// ForbiddenMessage 命中拒绝规则时的提示。
	// 默认："Forbidden"
	ForbiddenMessage string

	// TooManyRequestsCode 命中灰名单/限制造成 429 的业务错误码。
	// 默认："firewall_limited"
	TooManyRequestsCode string

	// TooManyRequestsMessage 命中灰名单/限制造成 429 的提示。
	// 默认："Too Many Requests"
	TooManyRequestsMessage string

	// SecurityEmitter 安全事件发射器（可选）。
	// 用于把 Firewall 命中沉淀为安全事件，便于后续封禁/告警/BI。
	SecurityEmitter *security.SecurityEventEmitter
}

func (c *FirewallConfig) withDefaults() FirewallConfig {
	cfg := *c
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 2 << 20 // 2MB
	}
	if cfg.MaxBodyBytesLarge <= 0 {
		cfg.MaxBodyBytesLarge = 20 << 20 // 20MB
	}
	if len(cfg.BlockUAKeywords) == 0 {
		cfg.BlockUAKeywords = []string{
			"sqlmap",
			"nikto",
			"acunetix",
			"netsparker",
			"masscan",
			"nmap",
			"zgrab",
			"whatweb",
			"wpscan",
			"fuzz",
			"dirbuster",
			"gobuster",
			"curl/", // 生产环境很多团队会允许 curl，这里只是默认示例，可按需删除
		}
	}
	if len(cfg.BlockPathSubstrings) == 0 {
		cfg.BlockPathSubstrings = []string{"..", "//", "\\"}
	}
	if cfg.ForbiddenCode == "" {
		cfg.ForbiddenCode = "firewall_forbidden"
	}
	if cfg.ForbiddenMessage == "" {
		cfg.ForbiddenMessage = "Forbidden"
	}
	if cfg.TooManyRequestsCode == "" {
		cfg.TooManyRequestsCode = "firewall_limited"
	}
	if cfg.TooManyRequestsMessage == "" {
		cfg.TooManyRequestsMessage = "Too Many Requests"
	}
	return cfg
}

// Firewall 轻量 WAF 中间件。
func Firewall(cfg FirewallConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	allowMethods := make(map[string]struct{}, len(cfg.AllowMethods))
	for _, m := range cfg.AllowMethods {
		mm := strings.ToUpper(strings.TrimSpace(m))
		if mm != "" {
			allowMethods[mm] = struct{}{}
		}
	}

	blockIPs := make(map[string]struct{}, len(cfg.IPBlockList))
	for _, ip := range cfg.IPBlockList {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			blockIPs[ip] = struct{}{}
		}
	}
	grayIPs := make(map[string]struct{}, len(cfg.IPGrayList))
	for _, ip := range cfg.IPGrayList {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			grayIPs[ip] = struct{}{}
		}
	}

	uaKeywords := make([]string, 0, len(cfg.BlockUAKeywords))
	for _, k := range cfg.BlockUAKeywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			uaKeywords = append(uaKeywords, k)
		}
	}

	blockPathSubs := make([]string, 0, len(cfg.BlockPathSubstrings))
	for _, s := range cfg.BlockPathSubstrings {
		s = strings.TrimSpace(s)
		if s != "" {
			blockPathSubs = append(blockPathSubs, s)
		}
	}

	return func(c *gin.Context) {
		// 1) 方法白名单
		if _, ok := allowMethods[c.Request.Method]; !ok {
			abortForbidden(c, cfg, "method_not_allowed", map[string]any{
				"method": c.Request.Method,
			})
			return
		}

		// 2) 解析客户端 IP（基于 RealIP 中间件）
		ip := GetClientIP(c)
		ip = strings.TrimSpace(ip)
		ip = stripPossibleIP(ip)

		// 3) IP 黑/灰名单
		if ip != "" {
			if _, ok := blockIPs[ip]; ok {
				abortForbidden(c, cfg, "ip_blocked", map[string]any{
					"ip": ip,
				})
				return
			}
			if _, ok := grayIPs[ip]; ok {
				abortTooMany(c, cfg, "ip_graylisted", map[string]any{
					"ip": ip,
				})
				return
			}
		}

		// 4) Path 基础过滤
		path := c.Request.URL.Path
		if path == "" {
			abortForbidden(c, cfg, "empty_path", nil)
			return
		}
		if containsControlChars(path) {
			abortForbidden(c, cfg, "path_control_chars", nil)
			return
		}
		for _, sub := range blockPathSubs {
			if strings.Contains(path, sub) {
				abortForbidden(c, cfg, "path_forbidden_substring", nil)
				return
			}
		}

		// 5) UA/Referer 规则
		ua := strings.TrimSpace(c.Request.UserAgent())
		if ua == "" {
			if cfg.RejectEmptyUA {
				abortForbidden(c, cfg, "empty_ua", nil)
				return
			}
		} else {
			lua := strings.ToLower(ua)
			for _, k := range uaKeywords {
				if strings.Contains(lua, k) {
					abortForbidden(c, cfg, "ua_blocked", nil)
					return
				}
			}
		}

		referer := strings.TrimSpace(c.Request.Referer())
		if referer == "" {
			if cfg.RejectEmptyReferer {
				abortForbidden(c, cfg, "empty_referer", nil)
				return
			}
		}

		// 6) BodySize 限制（使用 Content-Length 快速判断；无 Content-Length 则不在此阶段读取 body，交给后续/网关）
		maxBody := cfg.MaxBodyBytes
		if hasAnyPrefix(path, cfg.LargePathPrefixes) {
			maxBody = cfg.MaxBodyBytesLarge
		}
		if c.Request.ContentLength > maxBody {
			abortForbidden(c, cfg, "body_too_large", nil)
			return
		}

		// 7) Content-Type 校验（仅对 JSON 路由）
		if hasAnyPrefix(path, cfg.JsonPathPrefixes) {
			// 对 GET/DELETE 通常不需要 body，放行
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodDelete {
				ct := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
				if ct == "" || !isJSONContentType(ct) {
					abortForbidden(c, cfg, "invalid_content_type", nil)
					return
				}
				// 轻量校验：如果 body 非空且很小，尝试快速判断是否 JSON（避免明显垃圾 body）
				// 注意：不做深度解析。
				if c.Request.ContentLength > 0 && c.Request.ContentLength <= 1024 {
					b, ok := peekBody(c, 1024)
					if ok && len(bytes.TrimSpace(b)) > 0 {
						if !looksLikeJSON(b) {
							abortForbidden(c, cfg, "invalid_json_body", nil)
							return
						}
					}
				}
			}
		}

		c.Next()
	}
}

func abortForbidden(c *gin.Context, cfg FirewallConfig, reason string, details map[string]any) {
	// 先发安全事件
	emitFirewallSecurityEvent(c, cfg.SecurityEmitter, "firewall_block", security.SecuritySeverityHigh, reason, details)

	if c.Writer.Written() {
		c.Abort()
		return
	}
	rid := GetRequestID(c)
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"code": cfg.ForbiddenCode,
		"msg":  cfg.ForbiddenMessage,
		"rid":  rid,
	})
}

func abortTooMany(c *gin.Context, cfg FirewallConfig, reason string, details map[string]any) {
	// 先发安全事件
	emitFirewallSecurityEvent(c, cfg.SecurityEmitter, "firewall_limit", security.SecuritySeverityMedium, reason, details)

	if c.Writer.Written() {
		c.Abort()
		return
	}
	rid := GetRequestID(c)
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code": cfg.TooManyRequestsCode,
		"msg":  cfg.TooManyRequestsMessage,
		"rid":  rid,
	})
}

func hasAnyPrefix(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func containsControlChars(s string) bool {
	// 控制字符（除去常见可见字符）直接拒绝
	for _, r := range s {
		if r == 0 {
			return true
		}
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	// 非法 UTF-8 也拒绝
	return !utf8.ValidString(s)
}

func stripPossibleIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	// 防止传入 host:port
	if h, _, err := net.SplitHostPort(ip); err == nil {
		ip = h
	}
	// 去掉 IPv6 bracket
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")
	return ip
}

func isJSONContentType(ct string) bool {
	// 允许：application/json; charset=utf-8
	if strings.HasPrefix(ct, "application/json") {
		return true
	}
	// 兼容部分客户端
	if strings.HasPrefix(ct, "application/ld+json") {
		return true
	}
	return false
}

func looksLikeJSON(b []byte) bool {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return true
	}
	// 快速前导字符判断
	if b[0] != '{' && b[0] != '[' {
		return false
	}
	// 进一步轻量校验：尝试 json.Valid（不会生成结构体，开销较小）
	return json.Valid(b)
}

// peekBody 轻量窥探请求体前 N 字节。
// 注意：只适合小 body；不会触发大 body 读取。
// 读取后会把 body 复原。
func peekBody(c *gin.Context, n int64) ([]byte, bool) {
	if c.Request.Body == nil {
		return nil, false
	}
	// 保护：不要对未知/大 body 读取
	if c.Request.ContentLength < 0 || c.Request.ContentLength > n {
		return nil, false
	}

	data, err := io.ReadAll(io.LimitReader(c.Request.Body, n))
	if err != nil {
		return nil, false
	}
	// 复原 body
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	return data, true
}

// emitFirewallSecurityEvent 发送 Firewall 命中安全事件。
// 仅当 emitter 不为 nil 且配置了 Writer 或 Queue 时才发送。
// 自动补齐 details 中的 ip/path/method/ua 字段。
// reason 会写入 details["reason"]。
func emitFirewallSecurityEvent(c *gin.Context, emitter *security.SecurityEventEmitter, typ string, severity security.SecuritySeverity, reason string, details map[string]any) {
	if emitter == nil || (emitter.Writer == nil && emitter.Queue == nil) {
		return
	}
	if details == nil {
		details = map[string]any{}
	}
	if strings.TrimSpace(reason) != "" {
		details["reason"] = reason
	}
	// 兜底补齐常用字段
	if _, ok := details["ip"]; !ok {
		details["ip"] = GetClientIP(c)
	}
	if _, ok := details["path"]; !ok {
		details["path"] = c.Request.URL.Path
	}
	if _, ok := details["method"]; !ok {
		details["method"] = c.Request.Method
	}
	if _, ok := details["ua"]; !ok {
		details["ua"] = strings.TrimSpace(c.Request.UserAgent())
	}
	_ = security.EmitSecurityEvent(c, emitter, typ, severity, details)
}
