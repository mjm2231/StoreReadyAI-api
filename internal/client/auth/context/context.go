package context

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyClaims         = "auth_claims"
	ContextKeyUID            = "auth_uid"
	ContextKeyTenantID       = "auth_tenant_id"
	ContextKeyRole           = "auth_role"
	ContextKeyScopes         = "auth_scopes"
	ContextKeyTokenVer       = "auth_token_ver"
	ContextKeyRawToken       = "auth_raw_token"
	ContextKeyAuthType       = "auth_type"
	ContextKeyResolvedUserID = "auth_resolved_user_id"
)

// ClaimsSnapshot 是注入到 Gin context 的运行时身份快照。
//
// 说明：
// 1. 它不是 JWT 原始 claims，也不承担签名/解析职责；
// 2. 只用于 middleware 与 handler 之间传递当前请求身份信息；
// 3. 字段设计贴合当前 App 鉴权链路最常用读取需求。
type ClaimsSnapshot struct {
	UID            string
	TenantID       string
	Role           string
	Scopes         []string
	TokenVer       int64
	RawToken       string
	AuthType       string
	ResolvedUserID uint64
}

func SetClaims(c *gin.Context, claims ClaimsSnapshot, injectRawToken bool) {
	if c == nil {
		return
	}

	claims.UID = strings.TrimSpace(claims.UID)
	claims.TenantID = strings.TrimSpace(claims.TenantID)
	claims.Role = strings.TrimSpace(claims.Role)
	claims.AuthType = strings.TrimSpace(claims.AuthType)
	claims.RawToken = strings.TrimSpace(claims.RawToken)
	claims.Scopes = cloneStrings(claims.Scopes)

	c.Set(ContextKeyClaims, claims)
	c.Set(ContextKeyUID, claims.UID)
	c.Set(ContextKeyTenantID, claims.TenantID)
	c.Set(ContextKeyRole, claims.Role)
	c.Set(ContextKeyScopes, claims.Scopes)
	c.Set(ContextKeyTokenVer, claims.TokenVer)
	c.Set(ContextKeyAuthType, claims.AuthType)
	c.Set(ContextKeyResolvedUserID, claims.ResolvedUserID)
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
	claims.UID = strings.TrimSpace(claims.UID)
	claims.TenantID = strings.TrimSpace(claims.TenantID)
	claims.Role = strings.TrimSpace(claims.Role)
	claims.AuthType = strings.TrimSpace(claims.AuthType)
	claims.RawToken = strings.TrimSpace(claims.RawToken)
	claims.Scopes = cloneStrings(claims.Scopes)
	return claims, true
}

func GetUID(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyUID)
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

func GetTenantID(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyTenantID)
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

func GetRole(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyRole)
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

func GetScopes(c *gin.Context) []string {
	if c == nil {
		return nil
	}
	v, ok := c.Get(ContextKeyScopes)
	if !ok || v == nil {
		return nil
	}
	scopes, ok := v.([]string)
	if !ok {
		return nil
	}
	return cloneStrings(scopes)
}

func HasScope(c *gin.Context, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	for _, item := range GetScopes(c) {
		if item == scope {
			return true
		}
	}
	return false
}

func GetTokenVersion(c *gin.Context) (int64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(ContextKeyTokenVer)
	if !ok || v == nil {
		return 0, false
	}
	tokenVer, ok := v.(int64)
	if ok {
		return tokenVer, true
	}
	return 0, false
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

func SetResolvedUserID(c *gin.Context, userID uint64) {
	if c == nil {
		return
	}
	c.Set(ContextKeyResolvedUserID, userID)
	if claims, ok := GetClaims(c); ok {
		claims.ResolvedUserID = userID
		c.Set(ContextKeyClaims, claims)
	}
}

func GetResolvedUserID(c *gin.Context) (uint64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(ContextKeyResolvedUserID)
	if !ok || v == nil {
		return 0, false
	}
	userID, ok := v.(uint64)
	if !ok || userID == 0 {
		return 0, false
	}
	return userID, true
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
