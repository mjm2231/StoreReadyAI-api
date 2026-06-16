package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	rolehandler "storeready_ai/internal/admin/modules/roles/handler"
)

// registerRoleRoutes 注册后台角色模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/roles 下；
// 2. 进入本函数的 router group 默认应已完成后台登录鉴权；
// 3. 当前先按角色做最小权限控制，后续可平滑切到 permission codes。
func registerRoleRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.RoleHandler
	if h == nil {
		return
	}

	roles := r.Group("/roles")
	{
		roles.POST("/list",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.List,
		)
		roles.POST("/detail",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.Detail,
		)
		roles.POST("/create",
			adminmw.RequireSuperAdmin(),
			h.Create,
		)
		roles.POST("/update",
			adminmw.RequireSuperAdmin(),
			h.Update,
		)
		roles.POST("/status/update",
			adminmw.RequireSuperAdmin(),
			h.UpdateStatus,
		)
		roles.POST("/delete",
			adminmw.RequireSuperAdmin(),
			h.Delete,
		)
	}
}

var _ *rolehandler.Handler
