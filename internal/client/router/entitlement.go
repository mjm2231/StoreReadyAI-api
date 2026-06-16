package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterEntitlementRoutes registers entitlement/vip-related routes.
// MVP：VIP 权益状态查询（以及可选的手动开通/撤销，用于后台/调试）。
// 注意：这里采用显式接线（由 router.go 调用），不使用 init 自注册。
func RegisterEntitlementRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	vip := v1.Group("/vip")

	// Health for this module
	vip.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "vip pong"})
	})

	if d.EntitlementHandler == nil {
		panic("router: EntitlementHandler is nil (did you wire it in app layer?)")
	}

	// 需要登录
	api := vip.Group("/api", requireAppAuth(d))
	{
		// 统一使用 POST
		api.POST("/status", d.EntitlementHandler.Status)

		// 仅后台/调试（建议后续加更严格的 admin 鉴权）
		api.POST("/grant", d.EntitlementHandler.Grant)
		api.POST("/revoke", d.EntitlementHandler.Revoke)

		api.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "vip pong"})
		})
	}
}
