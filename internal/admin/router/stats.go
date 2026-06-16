package router

import (
	"github.com/gin-gonic/gin"

	adminmw "storeready_ai/internal/admin/middleware"
	statshandler "storeready_ai/internal/admin/modules/stats/handler"
)

// registerStatsRoutes 注册后台统计模块路由。
//
// 路由约定：
// 1. 当前统一挂在 /admin-api/stats 下；
// 2. 进入本函数的 router group 默认应已完成后台登录鉴权；
// 3. 当前先按角色做最小权限控制，后续可平滑切到 permission codes。
func registerStatsRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.StatsHandler
	if h == nil {
		return
	}

	stats := r.Group("/stats")
	{
		stats.POST("/overview",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetOverview,
		)
		stats.POST("/users/core",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetUserCoreStats,
		)
		stats.POST("/login/core",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetLoginStats,
		)
		stats.POST("/subscriptions/created-trend",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionCreatedTrend,
		)
		stats.POST("/subscriptions/cycle",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionCycleStats,
		)
		stats.POST("/subscriptions/currency",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionCurrencyStats,
		)
		stats.POST("/subscriptions/core",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionCoreStats,
		)
		stats.POST("/subscriptions/form",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionFormStats,
		)
		stats.POST("/subscriptions/sync",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetSubscriptionSyncStats,
		)
		stats.POST("/reminders/sent-trend",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetReminderSentTrend,
		)
		stats.POST("/reminders/status",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetReminderStatusStats,
		)
		stats.POST("/reminders/core",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetReminderCoreStats,
		)
		stats.POST("/reminders/settings",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetReminderSettingsStats,
		)
		stats.POST("/users/created-trend",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetUserCreatedTrend,
		)
		stats.POST("/overview/page",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetOverviewPageStats,
		)
		stats.POST("/vip/core",
			adminmw.RequireAnyRole("super_admin", "admin"),
			h.GetVipCoreStats,
		)
	}
}

var _ *statshandler.Handler
