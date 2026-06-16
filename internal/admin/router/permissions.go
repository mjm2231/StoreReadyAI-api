package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	permissionhandler "storeready_ai/internal/admin/modules/permissions/handler"
)

// registerPermissionRoutes 注册后台权限模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/permissions 下；
// 2. 进入本函数的 router group 默认应已完成后台登录鉴权；
// 3. 当前先按角色做最小权限控制，后续可平滑切到 permission codes。
func registerPermissionRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.PermissionHandler
	if h == nil {
		return
	}

	permissions := r.Group("/permissions")
	{
		permissions.POST("/list",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.List,
		)
		permissions.POST("/detail",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.Detail,
		)
		permissions.POST("/create",
			adminmw.RequireSuperAdmin(),
			h.Create,
		)
		permissions.POST("/update",
			adminmw.RequireSuperAdmin(),
			h.Update,
		)
		permissions.POST("/status/update",
			adminmw.RequireSuperAdmin(),
			h.UpdateStatus,
		)
		permissions.POST("/delete",
			adminmw.RequireSuperAdmin(),
			h.Delete,
		)
	}
}

var _ *permissionhandler.Handler
