package jwt

import (
	"strings"

	contractsauth "storeready_ai/internal/contracts/auth"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Claims 是 App 端 JWT 的标准 claims。
//
// 说明：
// 1. 只表达 token 自身携带的身份字段，不依赖 Gin / middleware；
// 2. 实现 contracts/auth.AppClaims，供 middleware/router 通过最小契约消费；
// 3. token_type 当前主要区分 access / refresh，默认由签发侧显式写入。
type Claims struct {
	UID       string   `json:"uid,omitempty"`
	TenantID  string   `json:"tenant_id,omitempty"`
	Role      string   `json:"role,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	TokenVer  int64    `json:"token_ver,omitempty"`
	TokenType string   `json:"token_type,omitempty"`

	jwtv5.RegisteredClaims
}

var _ contractsauth.AppClaims = (*Claims)(nil)

func (c *Claims) GetUID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.UID)
}

func (c *Claims) GetTenantID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.TenantID)
}

func (c *Claims) GetRole() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Role)
}

func (c *Claims) GetScopes() []string {
	if c == nil || len(c.Scopes) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.Scopes))
	for _, item := range c.Scopes {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *Claims) GetTokenVersion() int64 {
	if c == nil {
		return 0
	}
	return c.TokenVer
}

func (c *Claims) GetTokenType() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.TokenType)
}
