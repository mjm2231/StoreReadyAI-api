package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterSettingsRoutes registers settings-related routes.
// 用户全局设置（默认币种/提醒/时区/通知开关）。
// 注意：这里采用显式接线（由 router.go 调用），不使用 init 自注册。
// 说明：/api 路由均要求登录，tenant_id / user_id 来自认证上下文。
func RegisterSettingsRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	settings := v1.Group("/settings")

	// Health for this module
	settings.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "settings pong"})
	})

	if d.SettingsHandler == nil {
		panic("router: SettingsHandler is nil (did you wire it in app layer?)")
	}

	// 需要登录
	api := settings.Group("/api", requireAppAuth(d))
	{
		// 统一使用 POST
		api.POST("/get", d.SettingsHandler.Get)
		api.POST("/update", d.SettingsHandler.Update)
	}
}
