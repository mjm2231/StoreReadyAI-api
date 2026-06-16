package router

import (
	adminappuserhandler "storeready_ai/internal/admin/modules/appuser/handler"

	"github.com/gin-gonic/gin"
)

// registerAppUserRoutes 注册后台端“客户端用户”路由。
//
// 说明：
// 1. 这里的 user 指客户端 users，不是后台管理员 admin_users；
// 2. 路由统一挂在 admin 已鉴权分组下；
// 3. 当前先提供列表、详情、更新三个基础接口；
// 4. 若后续增加封禁/解封、VIP 调整、趋势统计等，再继续在此扩展。
func registerAppUserRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.AppUserHandler
	if h == nil {
		return
	}

	g := r.Group("/app-users")
	{
		g.POST("/list", h.ListUsers)
		g.POST("/get", h.GetUserByID)
		g.POST("/update", h.UpdateUser)
	}
}

var _ *adminappuserhandler.Handler
