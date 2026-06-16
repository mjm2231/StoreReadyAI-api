package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterDeviceRoutes registers device-related routes.
// MVP：设备登记/心跳/同步时间点（为多设备同步打基础）。
// 注意：这里采用显式接线（由 router.go 调用），不使用 init 自注册。
func RegisterDeviceRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	dev := v1.Group("/devices")

	// Health for this module
	dev.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "device pong"})
	})

	if d.DeviceHandler == nil {
		panic("router: DeviceHandler is nil (did you wire it in app layer?)")
	}

	// 需要登录
	api := dev.Group("/api", requireAppAuth(d))
	{
		// 统一使用 POST
		api.POST("/register", d.DeviceHandler.Register)
		api.POST("/heartbeat", d.DeviceHandler.Heartbeat)
		api.POST("/touch_sync", d.DeviceHandler.TouchSync)
		api.POST("/list", d.DeviceHandler.List)
		api.POST("/revoke", d.DeviceHandler.Revoke)

		api.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "device pong"})
		})
	}
}
