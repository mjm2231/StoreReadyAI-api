package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	authhandler "storeready_ai/internal/admin/modules/auth/handler"
)

// registerAuthRoutes 注册后台认证模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/auth 下；
// 2. 登录、刷新 token、注册属于公开或半公开接口，是否开放注册由上层权限策略决定；
// 3. 修改密码、退出登录、me 这类接口要求已完成后台登录鉴权。
func registerAuthRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.AuthHandler
	if h == nil {
		return
	}

	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh-token", h.RefreshToken)

		authed := auth.Group("")
		authed.Use(adminmw.RequireAuth(adminmw.AuthConfig{
			JWT: d.AdminJWT,
		}))
		{
			authed.POST("/logout", h.Logout)
			authed.POST("/change-password", h.ChangePassword)
		}
	}
}

var _ *authhandler.Handler
