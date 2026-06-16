package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterClientEventRoutes 注册客户端埋点路由。
//
// 路由说明：
// 1. 模块前缀统一使用 /v1/client-events；
// 2. 健康检查放在模块根路径下；
// 3. API 路由统一收口在 /api；
// 4. 当前单条上报、批量上报与查询都要求登录态，便于服务端补齐 tenant_id / uid。
func RegisterClientEventRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	ce := v1.Group("/client-events")

	// Health for this module
	ce.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "client-event pong"})
	})

	if d.ClientEventHandler == nil {
		panic("router: ClientEventHandler is nil (did you wire it in app layer?)")
	}

	// 需要登录
	api := ce.Group("/api")
	{
		// 统一使用 POST
		api.POST("/report", d.ClientEventHandler.Report)
		api.POST("/report-batch", d.ClientEventHandler.ReportBatch)
		api.POST("/list", d.ClientEventHandler.List)

		api.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "client-event pong"})
		})
	}
}
