package middleware

import (
	"net/http"
	"strings"

	authctx "storeready_ai/internal/admin/auth/context"
	contractsauth "storeready_ai/internal/contracts/auth"

	"github.com/gin-gonic/gin"
)

// AuthMode 表示后台管理员鉴权模式。
type AuthMode int

const (
	// AuthModeRequired 要求当前请求必须完成后台管理员鉴权。
	AuthModeRequired AuthMode = iota + 1
	// AuthModeOptional 表示当前请求可匿名访问；若携带合法后台身份则注入到 context。
	AuthModeOptional
)

// AuthClaims 是后台管理员运行时身份快照。
//
// 说明：
// 1. 它不是 JWT 原始 claims，而是 middleware 注入到 Gin context 的结构；
// 2. 当前直接复用 admin/auth/context 中定义的 ClaimsSnapshot，避免重复维护一份身份模型；
// 3. 为兼容现有调用方，这里保留 AuthClaims 名称作为类型别名。
type AuthClaims = authctx.ClaimsSnapshot

// AuthConfig 是后台管理员统一鉴权中间件配置。
//
// 设计说明：
// 1. required / optional 只是同一套鉴权逻辑的不同模式，而不是两套中间件实现；
// 2. middleware 只依赖 contracts/auth 中定义的 verifier 契约，不依赖具体 JWT 实现；
// 3. middleware 负责把解析结果映射为运行时 ClaimsSnapshot，并统一注入 Gin context；
// 4. 当前只处理 Bearer JWT，不在这里混入权限判断；权限判断应单独放到 permission middleware。
type AuthConfig struct {
	// Mode 鉴权模式。默认 required。
	Mode AuthMode

	// JWT 后台管理员 JWT 校验器。
	JWT contractsauth.AdminJWTVerifier

	// TokenExtractor Token 提取策略（可选）。
	// 若为空使用默认：Authorization Bearer -> X-Access-Token -> Cookie("admin_access_token")
	TokenExtractor func(c *gin.Context) string

	// 是否将 RawToken 写入 context（默认 false）
	InjectRawToken bool

	// 失败响应的业务码/提示
	RespCode    int16
	RespMessage string
}

func (c *AuthConfig) withDefaults() AuthConfig {
	cfg := *c
	if cfg.Mode == 0 {
		cfg.Mode = AuthModeRequired
	}
	if cfg.TokenExtractor == nil {
		cfg.TokenExtractor = defaultTokenExtractor
	}
	if cfg.RespCode == 0 {
		cfg.RespCode = 401
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Unauthorized"
	}
	return cfg
}

// Auth 统一后台管理员鉴权中间件。
func Auth(cfg AuthConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	if cfg.JWT == nil {
		panic("admin Auth: JWT verifier is required")
	}

	return func(c *gin.Context) {
		tok := strings.TrimSpace(cfg.TokenExtractor(c))
		if tok == "" {
			if cfg.Mode == AuthModeOptional {
				c.Next()
				return
			}
			abortUnauthorized(c, cfg, "missing_authorization")
			return
		}

		parsedClaims, err := cfg.JWT.ParseAccessToken(tok)
		if err != nil {
			if cfg.Mode == AuthModeOptional {
				c.Next()
				return
			}
			abortUnauthorized(c, cfg, "invalid_token")
			return
		}
		if parsedClaims == nil || parsedClaims.GetAdminUserID() == 0 || parsedClaims.GetTenantID() == 0 {
			if cfg.Mode == AuthModeOptional {
				c.Next()
				return
			}
			abortUnauthorized(c, cfg, "invalid_admin_identity")
			return
		}

		claims := AuthClaims{
			TenantID:    parsedClaims.GetTenantID(),
			AdminUserID: parsedClaims.GetAdminUserID(),
			Username:    strings.TrimSpace(parsedClaims.GetUsername()),
			Roles:       cloneRoles(parsedClaims.GetRoles()),
			RawToken:    tok,
			AuthType:    strings.TrimSpace(parsedClaims.GetTokenType()),
		}
		if claims.AuthType == "" {
			claims.AuthType = "jwt"
		}

		injectClaims(c, claims, cfg.InjectRawToken)
		c.Next()
	}
}

// RequireAuth 是 Auth(AuthModeRequired) 的语义包装。
func RequireAuth(cfg AuthConfig) gin.HandlerFunc {
	cfg.Mode = AuthModeRequired
	return Auth(cfg)
}

// OptionalAuth 是 Auth(AuthModeOptional) 的语义包装。
func OptionalAuth(cfg AuthConfig) gin.HandlerFunc {
	cfg.Mode = AuthModeOptional
	return Auth(cfg)
}

func injectClaims(c *gin.Context, claims AuthClaims, injectRaw bool) {
	authctx.SetClaims(c, claims, injectRaw)
}

func abortUnauthorized(c *gin.Context, cfg AuthConfig, code string) {
	if !c.Writer.Written() {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code": code,
			"msg":  cfg.RespMessage,
		})
		return
	}
	c.Abort()
}

func defaultTokenExtractor(c *gin.Context) string {
	az := strings.TrimSpace(c.GetHeader("Authorization"))
	if az != "" {
		parts := strings.SplitN(az, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	x := strings.TrimSpace(c.GetHeader("X-Access-Token"))
	if x != "" {
		return x
	}
	if v, err := c.Cookie("admin_access_token"); err == nil {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func cloneRoles(in []string) []string {
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
