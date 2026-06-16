package router

import (
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	adminhandler "storeready_ai/internal/admin/handler"
	adminrouter "storeready_ai/internal/admin/router"
	httphandler "storeready_ai/internal/client/http/handler"
	"storeready_ai/internal/client/middleware"
	feedbackhandler "storeready_ai/internal/client/modules/feedback/handler"
	projecthandler "storeready_ai/internal/client/modules/project/handler"
	userhandler "storeready_ai/internal/client/modules/user/handler"
	"storeready_ai/internal/config"
	contractsauth "storeready_ai/internal/contracts/auth"
	"storeready_ai/internal/i18n"
	audit "storeready_ai/internal/infra/audit"
	"storeready_ai/internal/infra/securityevent"
	"storeready_ai/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps Router 装配依赖。
//
// 设计原则：
//   - router 负责 HTTP 层装配（路由 + 中间件 + 内置端点）
//   - app/main 负责进程生命周期与基础资源创建
//
// 说明：
//   - Writer/Queue 可由上层传入；若为空且 DB 不为空，router 会创建默认的 GORM Writer
//   - Redis 可选：用于 RateLimit/AntiBrush（如果你的中间件配置需要）
type Deps struct {
	Cfg *config.Config
	// I18n 服务端多语言翻译器（错误提示/通知文案等动态文案）。
	I18n *i18n.Translator
	DB   *gorm.DB
	// Logger 应用主日志（zap）。用于 AccessLog 等中间件输出。
	Logger *zap.Logger

	// AuthHandler 认证相关 handler（登录/续期/登出）。
	AuthHandler *httphandler.AuthHandler

	// SettingsHandler 用户全局设置相关 handler（默认币种/提醒/时区/通知开关）。
	SettingsHandler *httphandler.SettingsHandler

	// DeviceHandler 设备相关 handler（登记/心跳/同步时间点/设备列表）。
	DeviceHandler *httphandler.DeviceHandler
	// BillingHandler 支付相关 handler（购买校验/恢复购买/权益查询/商品配置）。
	//
	// 说明：
	// 1. router 层只接已经装配完成的 BillingHandler；
	// 2. GooglePlay executor / GooglePlay client / verifier / billing service 的具体创建，
	//    应放在 app/bootstrap 组装层，不应放在 router；
	// 3. router 这里仅负责把 BillingHandler 挂到 HTTP 路由上。
	BillingHandler *httphandler.BillingHandler
	// EntitlementHandler VIP 权益相关 handler（状态查询/手动开通/撤销）。
	EntitlementHandler *httphandler.EntitlementHandler

	// ClientEventHandler 客户端埋点相关 handler（上报/查询）。
	ClientEventHandler *httphandler.ClientEventHandler
	FeedbackHandler    *feedbackhandler.Handler
	UserHandler        *userhandler.UserHandler
	ProjectHandler     *projecthandler.ProjectHandler

	Redis redis.UniversalClient

	// AuthJWT App 端 JWT 校验器（由上层 bootstrap 组装注入）。
	// router 只负责接线，不负责创建 JWT。
	AuthJWT contractsauth.AppJWTVerifier

	// AdminJWT 后台管理员 JWT 校验器（由上层 bootstrap 组装注入）。
	// 说明：
	// 1. Admin 与 App 端鉴权隔离，避免共用同一套 token/verifier；
	// 2. 使用公共 contracts/auth 中定义的 AdminJWTVerifier，保证编译期类型检查；
	// 3. router 只负责接线，不负责创建 Admin JWT。
	AdminJWT contractsauth.AdminJWTVerifier

	// AdminHandler Admin 通用 handler 聚合入口。
	// 说明：
	// 1. 当前先承接 ping/me 等最小骨架接口；
	// 2. 后续可继续往里注入 auth/user/role/permission/audit 等模块 handler；
	// 3. router 层只接线，不负责创建具体 admin 业务 handler。
	AdminHandler *adminhandler.Handler

	// UserIDResolver 可选：将鉴权 claims 中的 tenant_id + uid 解析为 users.id。
	//
	// 说明：
	// 1. 若配置，则 Auth / AuthOptional 在鉴权成功后会继续解析 user_id；
	// 2. 成功后会把 user_id 一并写入 gin.Context，供后续 handler / service 使用；
	// 3. router 只负责接线，不负责创建 resolver。
	UserIDResolver middleware.UserIDResolver

	AuditWriter    middleware.AuditWriter
	AuditQueue     middleware.AuditQueuePublisher
	SecurityWriter security.SecurityEventWriter
	SecurityQueue  security.SecurityEventQueuePublisher
}

// New 创建 Gin Engine，并完成中间件与基础路由装配。
func New(d Deps) *gin.Engine {
	// gin mode
	if d.Cfg != nil {
		m := strings.ToLower(strings.TrimSpace(d.Cfg.Server.Mode))
		if m == "release" {
			gin.SetMode(gin.ReleaseMode)
		} else {
			gin.SetMode(gin.DebugMode)
		}
	}

	r := gin.New()

	// 1) 依赖兜底：自动创建落库 writer（推荐结构：infra 实现）
	if d.AuditWriter == nil && d.DB != nil {
		d.AuditWriter = audit.NewGormAuditWriter(d.DB)
	}
	if d.SecurityWriter == nil && d.DB != nil {
		d.SecurityWriter = securityevent.NewGormSecurityEventWriter(d.DB)
	}

	// 2) 装配中间件（推荐顺序）
	bc := buildBootstrapConfig(d)
	r.Use(middleware.BootstrapMiddlewares(bc)...)

	// 3) 基础端点
	registerBaseRoutes(r, d)

	// 4) 业务路由（占位：后续在这里拆分模块路由）
	registerBizRoutes(r, d)

	return r
}

// buildBootstrapConfig 将配置映射到各中间件 Config。
func buildBootstrapConfig(d Deps) middleware.BootstrapConfig {

	bc := middleware.BootstrapConfig{}
	bptr := func(v bool) *bool { return &v }

	// 统一注入：审计/安全事件
	bc.AuditWriter = d.AuditWriter
	bc.AuditQueue = d.AuditQueue
	bc.SecurityWriter = d.SecurityWriter
	bc.SecurityQueue = d.SecurityQueue

	// AccessLog 默认可能使用 zap.L()；为保证写入 app.log，这里显式注入 app 层创建的 logger。
	if bc.AccessLog.Logger == nil && d.Logger != nil {
		// 确保 access log 开关不会被误关（未配置时默认 true；这里不强行覆盖 false）
		if bc.EnableAccessLog == nil {
			bc.EnableAccessLog = bptr(true)
		}

		bc.AccessLog.Logger = d.Logger
	}

	// Auth(JWT)
	// 说明：EnableAuth 默认值由 middleware 内部决定；这里在注入了 JWTVerifier 时显式开启。
	if d.AuthJWT != nil {
		bc.EnableAuth = bptr(true)
		bc.Auth.JWT = d.AuthJWT
		if d.UserIDResolver != nil {
			bc.Auth.UserIDResolver = d.UserIDResolver
		}
	}

	// 即使当前未配置 JWTVerifier，只要上层传入了 UserIDResolver，也继续挂到 AuthConfig 上。
	// 这样在某些仅使用 Session / 其他鉴权来源的场景下，AuthOptional 仍可在鉴权成功后解析 user_id。
	if d.AuthJWT == nil && d.UserIDResolver != nil {
		bc.Auth.UserIDResolver = d.UserIDResolver
	}

	// Redis 依赖（RateLimit/AntiBrush）
	// 说明：如果你的 RateLimit/AntiBrush Config 需要 Redis，这里统一注入
	if d.Redis != nil {
		bc.RateLimit.Redis = d.Redis
		bc.AntiBrush.Redis = d.Redis
	}

	// 根据配置映射 server.http.*
	if d.Cfg == nil {
		return bc
	}
	h := d.Cfg.Server.HTTP

	// RealIP
	bc.EnableRealIP = bptr(h.RealIP.Enabled)
	bc.RealIP.TrustedProxyCIDRs = h.RealIP.TrustedProxies
	// headers -> HeaderXFF/HeaderXRealIP/PreferXRealIP
	if len(h.RealIP.Headers) > 0 {
		bc.RealIP.HeaderXFF = h.RealIP.Headers[0]
	}
	if len(h.RealIP.Headers) > 1 {
		bc.RealIP.HeaderXRealIP = h.RealIP.Headers[1]
	}
	// 如果用户把 X-Real-IP 放在第一位，则优先使用 X-Real-IP
	if strings.EqualFold(strings.TrimSpace(bc.RealIP.HeaderXFF), "X-Real-IP") {
		bc.RealIP.PreferXRealIP = true
	}

	// Timeout（请求级）
	bc.Timeout.DefaultTimeout = h.Timeout.Default
	bc.Timeout.UseGatewayTimeout = h.Timeout.UseGatewayTimeout
	// per_route -> RouteTimeoutResolver
	if len(h.Timeout.PerRoute) > 0 {
		per := h.Timeout.PerRoute
		def := h.Timeout.Default
		bc.Timeout.RouteTimeoutResolver = func(c *gin.Context) time.Duration {
			if c == nil || c.Request == nil {
				return def
			}
			method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
			path := ""
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
			k1 := method + " " + path
			if v, ok := per[k1]; ok {
				return v
			}
			k2 := "* " + path
			if v, ok := per[k2]; ok {
				return v
			}
			return def
		}
	}

	// CORS
	bc.EnableCORS = bptr(h.CORS.Enabled)
	bc.CORS.AllowCredentials = h.CORS.AllowCredentials
	bc.CORS.AllowMethods = h.CORS.AllowMethods
	bc.CORS.AllowHeaders = h.CORS.AllowHeaders
	bc.CORS.ExposeHeaders = h.CORS.ExposeHeaders
	bc.CORS.MaxAge = h.CORS.MaxAge
	bc.CORS.AllowOrigins = h.CORS.AllowOrigins

	// Firewall
	bc.EnableFirewall = bptr(h.Firewall.Enabled)
	bc.Firewall.AllowMethods = h.Firewall.AllowMethods
	bc.Firewall.BlockPathSubstrings = h.Firewall.BlockPathSubstrings
	bc.Firewall.RejectEmptyUA = h.Firewall.RejectEmptyUA
	bc.Firewall.RejectEmptyReferer = h.Firewall.RejectEmptyReferer
	bc.Firewall.BlockUAKeywords = h.Firewall.BlockUAKeywords
	bc.Firewall.JsonPathPrefixes = h.Firewall.JSONPathPrefixes
	bc.Firewall.IPBlockList = h.Firewall.IPBlocklist
	bc.Firewall.IPGrayList = h.Firewall.IPGraylist
	// BodySize：普通/大包
	bc.Firewall.MaxBodyBytes = h.Limits.MaxBodyBytes
	bc.Firewall.MaxBodyBytesLarge = h.Limits.MaxBodyBytesLarge
	bc.Firewall.LargePathPrefixes = h.Limits.LargePathPrefixes

	// RateLimit
	bc.EnableRateLimit = bptr(h.RateLimit.Enabled)
	bc.RateLimit.PublicGetRPM = h.RateLimit.PublicGetRPM
	bc.RateLimit.WriteRPM = h.RateLimit.WriteRPM
	bc.RateLimit.AuthedRPM = h.RateLimit.AuthedRPM
	bc.RateLimit.PerRouteRPM = h.RateLimit.PerRoute

	// AntiBrush
	bc.EnableAntiBrush = bptr(h.AntiBrush.Enabled)
	// rules 配置化目前为占位：代码侧已内置默认规则，这里暂不解析

	return bc
}

// registerBaseRoutes 注册基础端点：health、metrics、pprof 等。
func registerBaseRoutes(r *gin.Engine, d Deps) {
	// health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "OK"})
	})

	// metrics
	if d.Cfg != nil && d.Cfg.Observability.Metrics.Enabled {
		path := strings.TrimSpace(d.Cfg.Observability.Metrics.Path)
		if path == "" {
			path = "/metrics"
		}
		// 可选 basic auth
		auth := d.Cfg.Observability.Metrics.Auth
		if auth.Enabled {
			r.GET(path, gin.BasicAuth(gin.Accounts{auth.Username: auth.Password}), middleware.MetricsHandler())
		} else {
			r.GET(path, middleware.MetricsHandler())
		}
	}

	// pprof
	if d.Cfg != nil && d.Cfg.Server.Pprof.Enabled {
		prefix := strings.TrimSpace(d.Cfg.Server.Pprof.PathPrefix)
		if prefix == "" {
			prefix = "/debug/pprof"
		}
		// 这里使用标准库 pprof 注册
		// 说明：pprof 建议仅在内网/调试开启
		registerPprof(r, prefix)
	}
}

// registerBizRoutes 业务路由（占位）。
func registerBizRoutes(r *gin.Engine, d Deps) {
	// 这里建议按模块拆分：
	//   - /v1/auth
	//   - /v1/subscriptions
	//   - /admin-api
	// 后续你可以把每个模块路由拆到独立文件：router/auth.go、router/admin.go ...

	v1 := r.Group("/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "pong"})
		})
	}
	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // 访问 /swagger/index.html。

	// App 端业务路由。
	// Admin 路由后续建议单独拆到 RegisterAdminRoutes，并挂在 /admin-api 下，
	// 同时使用独立的 AdminJWT / AdminAuth 中间件，避免与 App 鉴权链路耦合。
	RegisterAuthRoutes(r, d)

	RegisterSettingsRoutes(r, d)

	RegisterDeviceRoutes(r, d)
	// Billing 路由。
	//
	// 说明：
	// 1. 这里只负责注册路由；
	// 2. Billing 相关依赖（executor/client/verifier/service/handler）应在上层先装配完成；
	// 3. 若 d.BillingHandler 为空，具体 panic/兜底逻辑由 RegisterBillingRoutes 内部处理。
	RegisterBillingRoutes(r, d)
	RegisterEntitlementRoutes(r, d)

	RegisterClientEventRoutes(r, d)
	RegisterFeedbackRoutes(r, d)
	RegisterUserRoutes(r, d)
	RegisterProjectRoutes(r, d)

	registerAdminRoutes(r, d)
}

// registerAdminRoutes 注册 Admin 路由。
//
// 设计说明：
// 1. 当前 admin 与 app 共用同一个 gin.Engine，但走独立前缀 /admin-api；
// 2. AdminJWT 与 App AuthJWT 分离，避免后台 token 误入 App 鉴权链路；
// 3. 后续如果拆双 main 或独立 admin server，只需要把本函数迁移到 admin 入口调用即可。
func registerAdminRoutes(r *gin.Engine, d Deps) {
	if r == nil {
		return
	}
	if d.AdminJWT == nil {
		return
	}

	// Admin 端路由。
	// 当前统一注册在主 router 中，但使用独立前缀 /admin-api 和独立 AdminJWT 鉴权链路。
	adminrouter.RegisterRoutes(r, adminrouter.Deps{
		AdminJWT: d.AdminJWT,
		Handler:  d.AdminHandler,
	})
}

// registerPprof 注册 pprof 端点。
//
// 注意：
//   - 仅建议在内网/调试环境启用
//   - 如需更强控制（白名单/鉴权），可在此处加 Auth 中间件
func registerPprof(r *gin.Engine, prefix string) {
	g := r.Group(prefix)
	{
		g.GET("/", gin.WrapF(pprof.Index))
		g.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		g.GET("/profile", gin.WrapF(pprof.Profile))
		g.GET("/symbol", gin.WrapF(pprof.Symbol))
		g.POST("/symbol", gin.WrapF(pprof.Symbol))
		g.GET("/trace", gin.WrapF(pprof.Trace))

		// 常用 profiles
		g.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		g.GET("/block", gin.WrapH(pprof.Handler("block")))
		g.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		g.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		g.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		g.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}

func _appAuthConfig(d Deps) middleware.AuthConfig {
	return middleware.AuthConfig{
		JWT:            d.AuthJWT,
		UserIDResolver: d.UserIDResolver,
	}
}

func requireAppAuth(d Deps) gin.HandlerFunc {
	return middleware.RequireAuth(_appAuthConfig(d))
}

func optionalAppAuth(d Deps) gin.HandlerFunc {
	return middleware.OptionalAuth(_appAuthConfig(d))
}
