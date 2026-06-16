package middleware

import (
	"net"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

// 用于存储解析后的客户端IP的上下文键。
const CtxKeyClientIP = "client_ip"

// RealIPConfig 控制在代理后面如何解析真实客户端IP。
//
// 安全模型：
//   - 仅当直接连接的对端（RemoteAddr）位于 TrustedProxyCIDRs 中时，才信任转发头。
//   - 否则回退到对端IP（与 gin 的默认行为类似）。
//
// 注意：你仍然应该配置你的反向代理只设置以下之一：
// X-Forwarded-For / X-Real-IP，并避免客户端伪造它们。
// 此中间件通过只信任受信任代理的头部来保护你。
type RealIPConfig struct {
	// TrustedProxyCIDRs 是 CIDR 字符串（例如 "10.0.0.0/8", "192.168.0.0/16", "1.2.3.4/32"）
	// 表示你受信任的代理/负载均衡网络。
	TrustedProxyCIDRs []string

	// HeaderXFF 是转发链的头名称，默认值："X-Forwarded-For"。
	HeaderXFF string

	// HeaderXRealIP 是单个真实IP的头名称，默认值："X-Real-IP"。
	HeaderXRealIP string

	// PreferXRealIP 控制是否优先检查 X-Real-IP 而不是 X-Forwarded-For。
	// 默认值：false（优先 XFF）。
	PreferXRealIP bool
}

func (c *RealIPConfig) withDefaults() RealIPConfig {
	cfg := *c
	if cfg.HeaderXFF == "" {
		cfg.HeaderXFF = "X-Forwarded-For"
	}
	if cfg.HeaderXRealIP == "" {
		cfg.HeaderXRealIP = "X-Real-IP"
	}
	return cfg
}

// RealIP 使用受信任代理CIDR解析客户端IP并存储到 gin.Context。
//
// 使用场景：
//   - 基于客户端IP的限流
//   - 反刷/风险评分
//   - 审计日志
func RealIP(cfg RealIPConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	trusted := compileCIDRs(cfg.TrustedProxyCIDRs)

	return func(c *gin.Context) {
		peerIP := remoteIP(c.Request.RemoteAddr)
		resolved := peerIP

		// 仅当直接连接的对端位于受信任代理网络时，才信任转发头。
		if peerIP.IsValid() && isTrusted(peerIP, trusted) {
			if cfg.PreferXRealIP {
				if ip := parseSingleIPHeader(c.GetHeader(cfg.HeaderXRealIP)); ip.IsValid() {
					resolved = ip
				} else if ip := parseXFF(c.GetHeader(cfg.HeaderXFF), trusted); ip.IsValid() {
					resolved = ip
				}
			} else {
				if ip := parseXFF(c.GetHeader(cfg.HeaderXFF), trusted); ip.IsValid() {
					resolved = ip
				} else if ip := parseSingleIPHeader(c.GetHeader(cfg.HeaderXRealIP)); ip.IsValid() {
					resolved = ip
				}
			}
		}

		// 回退：如果全部失败，使用 gin 的 ClientIP() 作为最后手段。
		// 这样即使 RemoteAddr 在某些测试中为空，也能保持合理行为。
		if !resolved.IsValid() {
			if ip := parseSingleIPHeader(c.ClientIP()); ip.IsValid() {
				resolved = ip
			}
		}

		if resolved.IsValid() {
			c.Set(CtxKeyClientIP, resolved.String())
		}
		c.Next()
	}
}

// GetClientIP 返回上下文中解析的客户端IP（如果存在），否则返回 gin 的 ClientIP。
func GetClientIP(c *gin.Context) string {
	if v, ok := c.Get(CtxKeyClientIP); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			return s
		}
	}
	return c.ClientIP()
}

// --- 辅助函数 ---

type cidrList []netip.Prefix

func compileCIDRs(cidrs []string) cidrList {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, s := range cidrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			// 忽略无效CIDR以保持中间件健壮性
			continue
		}
		out = append(out, p)
	}
	return out
}

func isTrusted(ip netip.Addr, trusted cidrList) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(remoteAddr string) netip.Addr {
	// net.SplitHostPort 需要端口；但 RemoteAddr 可能只是一个IP。
	if host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil {
		return parseAddr(host)
	}
	return parseAddr(strings.TrimSpace(remoteAddr))
}

func parseAddr(s string) netip.Addr {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}
	}
	// 去除可能的 IPv6 区域标识符。
	if i := strings.IndexByte(s, '%'); i >= 0 {
		s = s[:i]
	}
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}
	}
	return ip
}

func parseSingleIPHeader(v string) netip.Addr {
	v = strings.TrimSpace(v)
	if v == "" {
		return netip.Addr{}
	}
	// 有时头部可能包含多个值；取第一个。
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return parseAddr(v)
}

// parseXFF 解析 X-Forwarded-For 并返回第一个非受信任代理的IP。
// 它假设仅当直接对端受信任时，该头才可信。
func parseXFF(xff string, trusted cidrList) netip.Addr {
	xff = strings.TrimSpace(xff)
	if xff == "" {
		return netip.Addr{}
	}
	parts := strings.Split(xff, ",")
	for _, p := range parts {
		ip := parseAddr(p)
		if !ip.IsValid() {
			continue
		}
		// 选取第一个非受信任的IP作为客户端IP。
		if !isTrusted(ip, trusted) {
			return ip
		}
	}
	// 如果全部都是受信任的（罕见情况），返回最左边的有效IP。
	for _, p := range parts {
		ip := parseAddr(p)
		if ip.IsValid() {
			return ip
		}
	}
	return netip.Addr{}
}
