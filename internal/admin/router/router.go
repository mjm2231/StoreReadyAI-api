package router

import (
	"github.com/gin-gonic/gin"

	handler "storeready_ai/internal/admin/handler"
	adminmw "storeready_ai/internal/admin/middleware"
	contractsauth "storeready_ai/internal/contracts/auth"
)

// Deps 是 admin 路由依赖。
// 当前先只放 AdminJWT，后续再按模块逐步补充 service / handler / logger / audit 等依赖。
type Deps struct {
	// AdminJWT 后台管理员 JWT 校验器。
	// 说明：
	// 1. 与 App 端 JWT 鉴权隔离；
	// 2. 由上层 bootstrap / app 装配后注入；
	// 3. 使用公共 contracts/auth 中定义的 AdminJWTVerifier，保证编译期类型检查；
	// 4. router 只负责接线，不负责创建 verifier。
	AdminJWT contractsauth.AdminJWTVerifier

	// Handler 后台公共基础 handler。
	// 当前用于承接 /admin-api/ping 等非业务模块基础接口。
	Handler *handler.Handler
}

// RegisterRoutes 注册后台管理端路由。
//
// 当前先落一个最小可运行骨架：
// - /admin-api/ping        无鉴权健康检查
// - /admin-api/auth/*      后台认证相关接口
// - /admin-api/*           其它业务模块接口通过 RequireAuth 保护
//
// 当前已开始按模块拆分：
// - registerAuthRoutes
// - registerUserRoutes
// - registerRoleRoutes
// - registerPermissionRoutes
// - registerAuditRoutes
// - registerSubscriptionRoutes
// - registerStatsRoutes
// - registerReminderRoutes
//
// 后续建议继续补齐：
func RegisterRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}
	h := d.Handler
	println("h is nil?", h == nil)
	if h == nil {
		h = handler.New()
	}
	admin := r.Group("/admin-api")
	{
		admin.GET("/ping", h.Ping)
		registerAuthRoutes(admin, d)
		authed := admin.Group("")
		authed.Use(adminmw.RequireAuth(adminmw.AuthConfig{
			JWT: d.AdminJWT,
		}))
		{
			authed.POST("/me", h.MeHandler.Me)
			registerUserRoutes(authed, d)
			registerRoleRoutes(authed, d)
			registerPermissionRoutes(authed, d)
			registerAuditRoutes(authed, d)

			registerStatsRoutes(authed, d)
			registerAppUserRoutes(authed, d)
			registerFeedbackRoutes(authed, d)
		}
	}
}
