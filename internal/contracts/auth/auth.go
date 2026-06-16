package auth

// AppClaims 是 App 端访问令牌解析后的最小能力集合。
//
// 说明：
// 1. 这里只定义中间件/路由依赖的最小读取能力，不绑定具体 JWT 实现；
// 2. 具体实现可由 internal/auth/jwt.Claims 提供；
// 3. 字段设计优先贴合当前 App 端已有 JWT 语义，避免为了“抽象统一”反向破坏现有链路；
// 4. 后续若接入别的 token 方案，只要实现这些方法即可复用现有鉴权链路。
type AppClaims interface {
	GetUID() string
	GetTenantID() string
	GetRole() string
	GetScopes() []string
	GetTokenVersion() int64
	GetTokenType() string
}

// AdminClaims 是 Admin 端访问令牌解析后的最小能力集合。
//
// 说明：
// 1. Admin 与 App 的身份字段不同，不强行复用同一个 claims 结构；
// 2. 这里只抽象中间件真正需要读取的字段；
// 3. 具体实现可由 internal/admin/modules/auth/jwt.Claims 提供。
type AdminClaims interface {
	GetTenantID() uint64
	GetAdminUserID() uint64
	GetUsername() string
	GetRoles() []string
	GetTokenType() string
}

// AppJWTVerifier 定义 App 端 access token 校验能力。
//
// 约束：
// - 仅用于 access token 解析；
// - 返回值使用 AppClaims 接口，避免上层依赖具体 jwt 包实现。
type AppJWTVerifier interface {
	ParseAccessToken(token string) (AppClaims, error)
}

// AdminJWTVerifier 定义 Admin 端 access token 校验能力。
//
// 约束：
// - 仅用于 admin access token 解析；
// - 返回值使用 AdminClaims 接口，避免 router/middleware 直接依赖 admin jwt 实现。
type AdminJWTVerifier interface {
	ParseAccessToken(token string) (AdminClaims, error)
}
