package context

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyClaims       = "admin_auth_claims"
	ContextKeyTenantID     = "admin_auth_tenant_id"
	ContextKeyAdminUserID  = "admin_auth_user_id"
	ContextKeyUsername     = "admin_auth_username"
	ContextKeyIsSuperAdmin = "admin_auth_is_super_admin"
	ContextKeyRoles        = "admin_auth_roles"
	ContextKeyRawToken     = "admin_auth_raw_token"
	ContextKeyAuthType     = "admin_auth_type"
)

// ClaimsSnapshot 是注入到 Gin context 的后台管理员运行时身份快照。
//
// 说明：
// 1. 它不是 JWT 原始 claims，也不承担签名/解析职责；
// 2. 只用于 admin middleware 与 handler 之间传递当前请求身份信息；
// 3. 字段设计贴合当前 Admin 鉴权链路最常用读取需求。
type ClaimsSnapshot struct {
	TenantID     uint64
	AdminUserID  uint64
	Username     string
	IsSuperAdmin bool
	Roles        []string
	RawToken     string
	AuthType     string
}

func SetClaims(c *gin.Context, claims ClaimsSnapshot, injectRawToken bool) {
	if c == nil {
		return
	}

	claims.Username = strings.TrimSpace(claims.Username)
	claims.RawToken = strings.TrimSpace(claims.RawToken)
	claims.AuthType = strings.TrimSpace(claims.AuthType)
	claims.Roles = cloneStrings(claims.Roles)

	c.Set(ContextKeyClaims, claims)
	c.Set(ContextKeyTenantID, claims.TenantID)
	c.Set(ContextKeyAdminUserID, claims.AdminUserID)
	c.Set(ContextKeyUsername, claims.Username)
	c.Set(ContextKeyIsSuperAdmin, claims.IsSuperAdmin)
	c.Set(ContextKeyRoles, claims.Roles)
	c.Set(ContextKeyAuthType, claims.AuthType)
	if injectRawToken {
		c.Set(ContextKeyRawToken, claims.RawToken)
	}
}

func GetClaims(c *gin.Context) (ClaimsSnapshot, bool) {
	if c == nil {
		return ClaimsSnapshot{}, false
	}
	v, ok := c.Get(ContextKeyClaims)
	if !ok || v == nil {
		return ClaimsSnapshot{}, false
	}
	claims, ok := v.(ClaimsSnapshot)
	if !ok {
		return ClaimsSnapshot{}, false
	}
	claims.Username = strings.TrimSpace(claims.Username)
	claims.RawToken = strings.TrimSpace(claims.RawToken)
	claims.AuthType = strings.TrimSpace(claims.AuthType)
	claims.Roles = cloneStrings(claims.Roles)
	return claims, true
}

func GetTenantID(c *gin.Context) (uint64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(ContextKeyTenantID)
	if !ok || v == nil {
		return 0, false
	}
	tenantID, ok := v.(uint64)
	if !ok || tenantID == 0 {
		return 0, false
	}
	return tenantID, true
}

func GetAdminUserID(c *gin.Context) (uint64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(ContextKeyAdminUserID)
	if !ok || v == nil {
		return 0, false
	}
	adminUserID, ok := v.(uint64)
	if !ok || adminUserID == 0 {
		return 0, false
	}
	return adminUserID, true
}

func GetUsername(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyUsername)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func IsSuperAdmin(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(ContextKeyIsSuperAdmin)
	if !ok || v == nil {
		return false
	}
	isSuperAdmin, ok := v.(bool)
	if !ok {
		return false
	}
	return isSuperAdmin
}

func GetRoles(c *gin.Context) []string {
	if c == nil {
		return nil
	}
	v, ok := c.Get(ContextKeyRoles)
	if !ok || v == nil {
		return nil
	}
	roles, ok := v.([]string)
	if !ok {
		return nil
	}
	return cloneStrings(roles)
}

func HasRole(c *gin.Context, role string) bool {
	role = strings.TrimSpace(role)
	if role == "" {
		return false
	}
	for _, item := range GetRoles(c) {
		if item == role {
			return true
		}
	}
	return false
}

func GetAuthType(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyAuthType)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func GetRawToken(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyRawToken)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
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
