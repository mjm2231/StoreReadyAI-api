package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	audithandler "storeready_ai/internal/admin/modules/audit/handler"
)

// registerAuditRoutes 注册后台审计模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/audit 下；
// 2. 进入本函数的 router group 默认应已完成后台登录鉴权；
// 3. 审计日志属于高敏感数据，当前统一限制为 super_admin 访问。
func registerAuditRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.AuditHandler
	if h == nil {
		return
	}

	audit := r.Group("/audit")
	{
		audit.POST("/create",
			adminmw.RequireSuperAdmin(),
			h.Create,
		)
		audit.POST("/detail",
			adminmw.RequireSuperAdmin(),
			h.Detail,
		)
		audit.POST("/list",
			adminmw.RequireSuperAdmin(),
			h.List,
		)
	}
}

var _ *audithandler.Handler
