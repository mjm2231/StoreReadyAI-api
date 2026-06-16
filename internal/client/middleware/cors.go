package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig 用于配置严格 CORS 白名单策略。
//
// 关键安全模型：
//   - 只对允许的 Origin 放行并回写 Access-Control-Allow-Origin
//   - Origin 不在白名单：直接拒绝（不回写 *）
//   - 支持通配符模式："https://*.xxx.com"（只匹配一个域名层级，且包含 scheme）
//
// 注意：
//   - 非浏览器调用通常不带 Origin，此时不做 CORS 校验，直接放行。
//   - 如果需要对无 Origin 的调用也做限制，应在 Firewall/鉴权层处理。
type CORSConfig struct {
	// AllowOrigins 允许的 Origin 白名单。
	// 支持：
	//   1) 精确匹配："https://app.xxx.com"
	//   2) 通配符："https://*.xxx.com"（只匹配一个子域：a.xxx.com，
	//      不匹配 a.b.xxx.com；也不匹配 http/https 不同 scheme）
	AllowOrigins []string

	// AllowMethods 允许的方法。
	// 默认：GET,POST,PUT,PATCH,DELETE,OPTIONS
	AllowMethods []string

	// AllowHeaders 允许的请求头。
	// 默认：Content-Type,Authorization,X-Request-Id
	AllowHeaders []string

	// ExposeHeaders 允许前端读取的响应头。
	// 默认：X-Request-Id
	ExposeHeaders []string

	// AllowCredentials 是否允许携带 Cookie/认证信息。
	// 后台管理通常需要 true。
	AllowCredentials bool

	// MaxAge 预检请求缓存时间。
	// 默认：12 小时
	MaxAge time.Duration

	// ForbiddenCode Origin 不在白名单时的业务错误码。
	// 默认："cors_forbidden"
	ForbiddenCode string

	// ForbiddenMessage Origin 不在白名单时的提示。
	// 默认："CORS Forbidden"
	ForbiddenMessage string
}

func (c *CORSConfig) withDefaults() CORSConfig {
	cfg := *c
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = []string{"Content-Type", "Authorization", "X-Request-Id"}
	}
	if len(cfg.ExposeHeaders) == 0 {
		cfg.ExposeHeaders = []string{"X-Request-Id"}
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 12 * time.Hour
	}
	if cfg.ForbiddenCode == "" {
		cfg.ForbiddenCode = "cors_forbidden"
	}
	if cfg.ForbiddenMessage == "" {
		cfg.ForbiddenMessage = "CORS Forbidden"
	}
	return cfg
}

// CORS 严格 CORS 中间件：Origin 不在白名单直接拒绝。
func CORS(cfg CORSConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	matcher := newOriginMatcher(cfg.AllowOrigins)

	allowMethods := strings.Join(normalizeTokens(cfg.AllowMethods), ", ")
	allowHeaders := strings.Join(normalizeTokens(cfg.AllowHeaders), ", ")
	exposeHeaders := strings.Join(normalizeTokens(cfg.ExposeHeaders), ", ")
	maxAgeSeconds := int(cfg.MaxAge / time.Second)

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		// 没有 Origin：通常为非浏览器/同源场景，不做 CORS 校验
		if origin == "" {
			c.Next()
			return
		}

		allowed := matcher.Match(origin)
		triggeringPreflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""

		if !allowed {
			// 严格策略：直接拒绝，不返回 *
			if !c.Writer.Written() {
				rid := GetRequestID(c)
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code": cfg.ForbiddenCode,
					"msg":  cfg.ForbiddenMessage,
					"rid":  rid,
				})
				return
			}
			c.Abort()
			return
		}

		// 命中白名单：回写 CORS 响应头
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Vary", appendVary(h.Get("Vary"), "Origin"))
		h.Set("Access-Control-Allow-Methods", allowMethods)
		h.Set("Access-Control-Allow-Headers", allowHeaders)
		h.Set("Access-Control-Expose-Headers", exposeHeaders)
		h.Set("Access-Control-Max-Age", itoa(maxAgeSeconds))
		if cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}

		// 预检请求直接返回
		if triggeringPreflight {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ----------------------
// Origin 匹配器（支持精确 + 通配符）
// ----------------------

type originMatcher struct {
	exact    map[string]struct{}
	wildcard []originWildcard
}

type originWildcard struct {
	scheme string
	suffix string // 例如：.xxx.com
}

func newOriginMatcher(allow []string) *originMatcher {
	m := &originMatcher{exact: make(map[string]struct{})}
	for _, raw := range allow {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		// 尝试解析为 URL，确保包含 scheme + host
		u, err := url.Parse(s)
		if err == nil && u.Scheme != "" && u.Host != "" {
			// 通配符：host 以 "*." 开头
			if strings.HasPrefix(u.Host, "*.") {
				suffix := strings.TrimPrefix(u.Host, "*.")
				if suffix != "" {
					m.wildcard = append(m.wildcard, originWildcard{
						scheme: strings.ToLower(u.Scheme),
						suffix: "." + strings.ToLower(suffix),
					})
					continue
				}
			}
			// 精确匹配：按原字符串存储（保持大小写一致性，这里统一 lower）
			m.exact[strings.ToLower(s)] = struct{}{}
			continue
		}
		// 解析失败：退化为字符串规则（仍然支持 "https://*.xxx.com"）
		ls := strings.ToLower(s)
		m.exact[ls] = struct{}{}
	}
	return m
}

func (m *originMatcher) Match(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	lo := strings.ToLower(origin)
	if _, ok := m.exact[lo]; ok {
		return true
	}

	// 解析 Origin
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(stripPort(u.Host))

	for _, w := range m.wildcard {
		if scheme != w.scheme {
			continue
		}
		// 必须以 suffix 结尾，且前面只有一个 label（避免 *.a.b.com 过宽）
		if strings.HasSuffix(host, w.suffix) {
			prefix := strings.TrimSuffix(host, w.suffix)
			// prefix 必须形如 "sub"（不含点）且非空
			if prefix != "" && !strings.Contains(prefix, ".") {
				return true
			}
		}
	}
	return false
}

func stripPort(hostport string) string {
	// IPv6 形如 [::1]:443
	if strings.HasPrefix(hostport, "[") {
		if i := strings.Index(hostport, "]"); i >= 0 {
			// 取出 bracket 内的地址
			return hostport[1:i]
		}
	}
	// 普通 host:port
	if i := strings.LastIndex(hostport, ":"); i > 0 {
		// 如果右侧是数字，认为是端口
		p := hostport[i+1:]
		allDigits := true
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return hostport[:i]
		}
	}
	return hostport
}

func normalizeTokens(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		s := strings.TrimSpace(v)
		if s == "" {
			continue
		}
		su := http.CanonicalHeaderKey(s)
		// 方法名不要 CanonicalHeaderKey（会把 GET 变 Get），所以特殊处理
		if isHTTPMethodToken(s) {
			su = strings.ToUpper(s)
		}
		if _, ok := seen[su]; ok {
			continue
		}
		seen[su] = struct{}{}
		out = append(out, su)
	}
	return out
}

func isHTTPMethodToken(s string) bool {
	ss := strings.ToUpper(strings.TrimSpace(s))
	switch ss {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return true
	default:
		return false
	}
}

func appendVary(existing, v string) string {
	if existing == "" {
		return v
	}
	// 简单去重
	parts := strings.Split(existing, ",")
	for _, p := range parts {
		if strings.EqualFold(strings.TrimSpace(p), v) {
			return existing
		}
	}
	return existing + ", " + v
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [32]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}
