package security

import "strings"

// MaskPhone 手机号脱敏：保留前 3 后 4。
func MaskPhone(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 7 {
		return "***"
	}
	return s[:3] + "****" + s[len(s)-4:]
}

// MaskEmail 邮箱脱敏：保留前 2 + 域名。
func MaskEmail(s string) string {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, "@")
	if idx <= 1 {
		return "***"
	}
	name := s[:idx]
	domain := s[idx:]
	if len(name) <= 2 {
		return name[:1] + "***" + domain
	}
	return name[:2] + "***" + domain
}

// MaskToken token 脱敏：保留前 4 后 4。
func MaskToken(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}
