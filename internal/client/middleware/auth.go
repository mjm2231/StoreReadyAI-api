package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	authctx "storeready_ai/internal/client/auth/context"
	"storeready_ai/internal/common"
	contractsauth "storeready_ai/internal/contracts/auth"

	"github.com/gin-gonic/gin"
)

// AuthMode 表示鉴权模式。
type AuthMode int

const (
	// AuthModeRequired 要求当前请求必须完成鉴权。
	AuthModeRequired AuthMode = iota + 1
	// AuthModeOptional 表示当前请求可匿名访问；若携带合法身份则注入到 context。
	AuthModeOptional
)

// AuthClaims 是运行时身份快照。
//
// 说明：
// 1. 它不是 JWT 原始 claims，而是 middleware 注入到 Gin context 的结构；
// 2. 当前直接复用 auth/context 中定义的 ClaimsSnapshot，避免重复维护一份身份模型；
// 3. middleware 负责把解析结果映射为运行时 ClaimsSnapshot，并统一注入 Gin context；
// 4. 如果配置了 UserIDResolver，会在中间件层统一把 claims.UID 解析为 users.id，并注入 user_id；
// 5. JWT 优先，Session 次之；若请求中显式携带了非法 JWT，则不会再回退 Session，避免绕过。
type AuthClaims = authctx.ClaimsSnapshot

// SessionResolver 负责根据 session id 查找登录态信息。
//
// 返回：
//   - claims: 命中的运行时身份快照
//   - ok=false: session 不存在或已失效
//   - err!=nil: 存储/网络错误（默认按未登录处理）
type SessionResolver func(sessionID string) (claims AuthClaims, ok bool, err error)

// TokenVersionResolver 用于校验 tokenVersion。
//
// 当 claims.TokenVer > 0 时，会调用该函数取当前版本：
//   - 若 current != claims.TokenVer => 认为 token 失效（例如用户改密/登出全部设备）
//
// 返回：
//   - current：当前有效版本
//   - ok：是否能查到（查不到按不通过处理，避免放行）
//   - err：存储错误（默认按不通过处理，避免放行）
type TokenVersionResolver func(uid string) (current int64, ok bool, err error)

// UserIDResolver 用于把 claims 中的业务 uid 解析为 users 表内部主键。
//
// user_id 解析（可选）。
//
// 建议在 requireAppAuth / AuthConfig 初始化时注入，统一在中间件层完成 uid -> users.id 转换。
// 后续 handler/service 只从 context 读取 user_id，不要在每个 service 重复 resolve。
type UserIDResolver func(ctx context.Context, tenantID, uid string) (userID uint64, ok bool, err error)

// AuthConfig 是统一鉴权中间件配置。
//
// 设计说明：
// 1. required / optional 只是同一套鉴权逻辑的不同模式，而不是两套中间件实现；
// 2. middleware 只依赖 contracts/auth 中定义的 verifier 契约，不依赖具体 JWT 实现；
// 3. middleware 负责把解析结果映射为运行时 ClaimsSnapshot，并统一注入 Gin context；
// 4. 如果配置了 UserIDResolver，会在中间件层统一把 claims.UID 解析为 users.id，并注入 user_id；
// 5. JWT 优先，Session 次之；若请求中显式携带了非法 JWT，则不会再回退 Session，避免绕过。
type AuthConfig struct {
	// Mode 鉴权模式。默认 required。
	Mode AuthMode

	// 白名单：匹配 URL.Path 前缀直接放行（例如 /health、/auth/login）
	WhitelistPathPrefixes []string

	// 可选：精确匹配 gin.FullPath（例如 "/auth/login"），命中即放行
	WhitelistFullPaths map[string]struct{}

	// JWT 校验器（必选其一：JWT 或 Session）
	JWT contractsauth.AppJWTVerifier

	// Session 解析器（必选其一：JWT 或 Session）
	Session SessionResolver

	// tokenVersion 校验（可选）
	TokenVersionResolver TokenVersionResolver

	// user_id 解析（可选）。
	//
	// 建议在 requireAppAuth / AuthConfig 初始化时注入，统一在中间件层完成 uid -> users.id 转换。
	// 后续 handler/service 只从 context 读取 user_id，不要在每个 service 重复 resolve。
	UserIDResolver UserIDResolver

	// Token 提取策略（可选）：自定义从请求中获取 token。
	// 若为空使用默认：Authorization Bearer -> X-Access-Token -> Cookie("access_token")
	TokenExtractor func(c *gin.Context) string

	// SessionID 提取策略（可选）：自定义从请求中获取 session id。
	// 若为空使用默认：X-Session-Id -> Cookie("sid")
	SessionIDExtractor func(c *gin.Context) string

	// 是否将 claims.RawToken 写入 ctx（默认 false）
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
	if cfg.WhitelistFullPaths == nil {
		cfg.WhitelistFullPaths = map[string]struct{}{}
	}
	if cfg.TokenExtractor == nil {
		cfg.TokenExtractor = defaultTokenExtractor
	}
	if cfg.SessionIDExtractor == nil {
		cfg.SessionIDExtractor = defaultSessionIDExtractor
	}
	if cfg.RespCode == 0 {
		cfg.RespCode = 401
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Unauthorized"
	}
	return cfg
}

// Auth 统一鉴权中间件。
func Auth(cfg AuthConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	if cfg.JWT == nil && cfg.Session == nil {
		panic("Auth: JWT 与 Session 至少配置一个")
	}

	return func(c *gin.Context) {
		if isWhitelisted(c, cfg) {
			c.Next()
			return
		}

		// 1) 优先 JWT。
		if cfg.JWT != nil {
			tok := strings.TrimSpace(cfg.TokenExtractor(c))
			if tok != "" {
				parsedClaims, err := cfg.JWT.ParseAccessToken(tok)
				if err == nil {
					claims := AuthClaims{
						UID:      strings.TrimSpace(parsedClaims.GetUID()),
						TenantID: strings.TrimSpace(parsedClaims.GetTenantID()),
						Role:     strings.TrimSpace(parsedClaims.GetRole()),
						Scopes:   cloneScopes(parsedClaims.GetScopes()),
						TokenVer: parsedClaims.GetTokenVersion(),
						RawToken: tok,
						AuthType: strings.TrimSpace(parsedClaims.GetTokenType()),
					}
					if claims.AuthType == "" {
						claims.AuthType = "jwt"
					}
					if cfg.checkTokenVersion(&claims) && cfg.resolveAndInjectUserID(c, &claims) {
						injectClaims(c, claims, cfg.InjectRawToken)
						c.Next()
						return
					}
					if cfg.Mode == AuthModeOptional {
						c.Next()
						return
					}
					abortUnauthorized(c, cfg)
					return
				}

				// 请求中显式携带了 JWT，但解析失败：
				// required => 401；optional => 匿名继续，但不降级到 Session。
				if cfg.Mode == AuthModeOptional {
					c.Next()
					return
				}
				abortUnauthorized(c, cfg)
				return
			}
		}

		// 2) 再尝试 Session（仅在未携带 JWT 时才进入这里）。
		if cfg.Session != nil {
			sid := strings.TrimSpace(cfg.SessionIDExtractor(c))
			if sid != "" {
				claims, ok, err := cfg.Session(sid)
				if err == nil && ok {
					claims.AuthType = "session"
					if cfg.checkTokenVersion(&claims) && cfg.resolveAndInjectUserID(c, &claims) {
						injectClaims(c, claims, cfg.InjectRawToken)
						c.Next()
						return
					}
					if cfg.Mode == AuthModeOptional {
						c.Next()
						return
					}
					abortUnauthorized(c, cfg)
					return
				}
				if cfg.Mode == AuthModeOptional {
					c.Next()
					return
				}
			}
		}

		if cfg.Mode == AuthModeOptional {
			c.Next()
			return
		}
		abortUnauthorized(c, cfg)
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

func isWhitelisted(c *gin.Context, cfg AuthConfig) bool {
	fp := c.FullPath()
	if fp != "" {
		if _, ok := cfg.WhitelistFullPaths[fp]; ok {
			return true
		}
	}
	p := ""
	if c.Request != nil && c.Request.URL != nil {
		p = c.Request.URL.Path
	}
	if p == "" {
		return false
	}
	for _, pre := range cfg.WhitelistPathPrefixes {
		if pre == "" {
			continue
		}
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

func (cfg AuthConfig) checkTokenVersion(claims *AuthClaims) bool {
	if cfg.TokenVersionResolver == nil {
		return true
	}
	if claims == nil {
		return false
	}
	if strings.TrimSpace(claims.UID) == "" {
		return false
	}
	if claims.TokenVer <= 0 {
		return true
	}
	cur, ok, err := cfg.TokenVersionResolver(strings.TrimSpace(claims.UID))
	if err != nil || !ok {
		return false
	}
	return cur == claims.TokenVer
}

func (cfg AuthConfig) resolveAndInjectUserID(c *gin.Context, claims *AuthClaims) bool {
	if cfg.UserIDResolver == nil {
		return true
	}
	if claims == nil {
		return false
	}

	tenantID := strings.TrimSpace(claims.TenantID)
	uid := strings.TrimSpace(claims.UID)
	if tenantID == "" || uid == "" {
		return false
	}

	userID, ok, err := cfg.UserIDResolver(c.Request.Context(), tenantID, uid)
	if err != nil || !ok || userID == 0 {
		return false
	}

	claims.ResolvedUserID = userID
	return true
}

func cloneScopes(in []string) []string {
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

func injectClaims(c *gin.Context, claims AuthClaims, injectRaw bool) {
	// 新方案：统一写入 auth/context。
	authctx.SetClaims(c, claims, injectRaw)
	if claims.ResolvedUserID > 0 {
		authctx.SetResolvedUserID(c, claims.ResolvedUserID)
		injectResolvedUserID(c, claims.ResolvedUserID)
	}

	// 兼容现有 common/ctx 读取链路，避免一次性改动过大。
	common.SetUID(c, claims.UID)
	common.SetTenantID(c, claims.TenantID)
	if claims.Role != "" {
		common.SetRole(c, claims.Role)
	}
	if len(claims.Scopes) > 0 {
		common.SetScopes(c, claims.Scopes)
	}
	common.SetAuthType(c, claims.AuthType)
	if injectRaw {
		common.SetAuthRawToken(c, claims.RawToken)
	}
	if claims.TokenVer > 0 {
		common.SetTokenVersion(c, claims.TokenVer)
	}
}

func injectResolvedUserID(c *gin.Context, userID uint64) {
	if userID == 0 {
		return
	}

	// 统一注入 users.id。后续 handler/service 只读取 user_id，不再重复 uid -> user_id resolve。
	c.Set("user_id", userID)
	c.Set("userID", userID)
	c.Set("UserID", userID)
}

func abortUnauthorized(c *gin.Context, cfg AuthConfig) {
	if !c.Writer.Written() {
		rid := GetRequestID(c)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code": cfg.RespCode,
			"msg":  cfg.RespMessage,
			"rid":  rid,
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
	if v, err := c.Cookie("access_token"); err == nil {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func defaultSessionIDExtractor(c *gin.Context) string {
	sid := strings.TrimSpace(c.GetHeader("X-Session-Id"))
	if sid != "" {
		return sid
	}
	if v, err := c.Cookie("sid"); err == nil {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

// ErrUnauthorized 外部实现 SessionResolver 时可复用的错误。
var ErrUnauthorized = errors.New("unauthorized")
