package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// =====================
// Gin Context Keys
// =====================
// 目的：统一管理 gin.Context 中使用的 key，避免散落字符串导致的拼写错误。
// 约定：中间件写入，业务层只通过本文件的 helper 读取。

const (
	CtxKeyRID     = "rid"
	CtxKeyTraceID = "trace_id"
	CtxKeySpanID  = "span_id"
	CtxKeyIP      = "ip"
	CtxKeyUA      = "ua"
	CtxKeyReferer = "referer"

	CtxKeyUID        = "uid"
	CtxKeyTenantID   = "tenant_id"
	CtxKeyUserID     = "user_id"
	CtxKeyRole       = "role"
	CtxKeyScopes     = "scopes"
	CtxKeyAuthType   = "auth_type"
	CtxKeyAuthRawTok = "auth_raw_token"
	CtxKeyTokenVer   = "token_version"
)

// =====================
// Auth: setters (middleware)
// =====================

func SetUID(c *gin.Context, uid string) {
	c.Set(CtxKeyUID, uid)
}

func SetTenantID(c *gin.Context, tenantID string) {
	c.Set(CtxKeyTenantID, tenantID)
}

func SetUserID(c *gin.Context, userID uint64) {
	c.Set(CtxKeyUserID, userID)
}

func SetRole(c *gin.Context, role string) {
	c.Set(CtxKeyRole, role)
}

func SetScopes(c *gin.Context, scopes []string) {
	c.Set(CtxKeyScopes, scopes)
}

func SetAuthType(c *gin.Context, authType string) {
	c.Set(CtxKeyAuthType, authType)
}

func SetAuthRawToken(c *gin.Context, tok string) {
	c.Set(CtxKeyAuthRawTok, tok)
}

func SetTokenVersion(c *gin.Context, ver int64) {
	c.Set(CtxKeyTokenVer, ver)
}

// =====================
// Auth: getters (handler/service)
// =====================

func GetUID(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxKeyUID)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func MustUID(c *gin.Context) string {
	s, _ := GetUID(c)
	return s
}

func GetTenantID(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxKeyTenantID)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func MustTenantID(c *gin.Context) string {
	s, _ := GetTenantID(c)
	return s
}

func GetUserID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(CtxKeyUserID)
	if !ok || v == nil {
		return 0, false
	}

	switch t := v.(type) {
	case uint64:
		return t, t > 0
	case uint:
		return uint64(t), t > 0
	case int64:
		if t <= 0 {
			return 0, false
		}
		return uint64(t), true
	case int:
		if t <= 0 {
			return 0, false
		}
		return uint64(t), true
	case int32:
		if t <= 0 {
			return 0, false
		}
		return uint64(t), true
	case float64:
		if t <= 0 {
			return 0, false
		}
		return uint64(t), true
	case string:
		if t == "" {
			return 0, false
		}
		n, err := strconv.ParseUint(t, 10, 64)
		if err != nil || n == 0 {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func MustUserID(c *gin.Context) uint64 {
	n, _ := GetUserID(c)
	return n
}

func GetRole(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxKeyRole)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func GetScopes(c *gin.Context) ([]string, bool) {
	v, ok := c.Get(CtxKeyScopes)
	if !ok || v == nil {
		return nil, false
	}
	a, ok := v.([]string)
	return a, ok
}

func GetAuthType(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxKeyAuthType)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func GetAuthRawToken(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxKeyAuthRawTok)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func GetTokenVersion(c *gin.Context) (int64, bool) {
	v, ok := c.Get(CtxKeyTokenVer)
	if !ok || v == nil {
		return 0, false
	}

	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case float64:
		// 某些 JSON/Map 反序列化可能会变成 float64
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
