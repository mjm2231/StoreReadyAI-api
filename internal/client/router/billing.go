package router

import (
	"github.com/gin-gonic/gin"
)

func RegisterBillingRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	billing := v1.Group("/billing")

	// Health for this module
	billing.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "msg": "billing pong"})
	})

	if d.BillingHandler == nil {
		panic("router: BillingHandler is nil (did you wire it in app layer?)")
	}

	// 需要登录
	api := billing.Group("/api", requireAppAuth(d))
	{
		// 统一使用 POST
		api.POST("/verify", d.BillingHandler.VerifyPurchase)
		api.POST("/restore", d.BillingHandler.RestorePurchase)
		api.POST("/entitlement", d.BillingHandler.GetEntitlement)
		api.POST("/config", d.BillingHandler.GetConfig)

		api.POST("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "msg": "billing pong"})
		})
	}
}
