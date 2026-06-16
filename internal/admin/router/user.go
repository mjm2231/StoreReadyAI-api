package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	userhandler "storeready_ai/internal/admin/modules/user/handler"
)

// registerUserRoutes 注册后台管理员用户模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/users 下；
// 2. 进入本函数的 router group 默认应已完成后台登录鉴权；
// 3. 当前先按角色做最小权限控制，后续可平滑切到 permission codes。
func registerUserRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.UserHandler
	if h == nil {
		return
	}

	users := r.Group("/users")
	{
		users.POST("/list",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.List,
		)
		users.POST("/detail",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.Detail,
		)
		users.POST("/create",
			adminmw.RequireSuperAdmin(),
			h.Create,
		)
		users.POST("/update",
			adminmw.RequireSuperAdmin(),
			h.Update,
		)
		users.POST("/password/update",
			adminmw.RequireSuperAdmin(),
			h.UpdatePassword,
		)
		users.POST("/status/update",
			adminmw.RequireSuperAdmin(),
			h.UpdateStatus,
		)
		users.POST("/delete",
			adminmw.RequireSuperAdmin(),
			h.Delete,
		)
	}
}

var _ *userhandler.Handler
