package jwt

import (
	"strings"

	contractsauth "storeready_ai/internal/contracts/auth"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Claims 是 Admin 端 JWT 的标准 claims。
//
// 说明：
// 1. 只表达后台管理员 token 自身携带的身份字段，不依赖 Gin / middleware；
// 2. 实现 contracts/auth.AdminClaims，供 admin middleware/router 通过最小契约消费；
// 3. token_type 当前主要区分 access / refresh，默认由签发侧显式写入。
type Claims struct {
	TenantID    uint64   `json:"tenant_id,omitempty"`
	AdminUserID uint64   `json:"admin_user_id,omitempty"`
	Username    string   `json:"username,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	TokenType   string   `json:"token_type,omitempty"`

	jwtv5.RegisteredClaims
}

var _ contractsauth.AdminClaims = (*Claims)(nil)

func (c *Claims) GetTenantID() uint64 {
	if c == nil {
		return 0
	}
	return c.TenantID
}

func (c *Claims) GetAdminUserID() uint64 {
	if c == nil {
		return 0
	}
	return c.AdminUserID
}

func (c *Claims) GetUsername() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Username)
}

func (c *Claims) GetRoles() []string {
	if c == nil || len(c.Roles) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.Roles))
	for _, item := range c.Roles {
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

func (c *Claims) GetTokenType() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.TokenType)
}
