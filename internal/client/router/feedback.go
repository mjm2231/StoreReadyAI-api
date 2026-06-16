package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterFeedbackRoutes registers feedback-related routes.
// MVP：用户反馈提交 + 后台反馈管理。
// 注意：这里采用显式接线（由 router.go 调用），不使用 init 自注册。
func RegisterFeedbackRoutes(r *gin.Engine, d Deps) {
	if d.FeedbackHandler == nil {
		panic("router: FeedbackHandler is nil (did you wire it in app layer?)")
	}

	// Client feedback routes.
	v1 := r.Group("/v1")
	feedback := v1.Group("/feedback")

	// Health for this module.
	feedback.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "feedback pong"})
	})

	// App 端反馈接口。
	// 说明：创建反馈通常需要登录，便于关联 uid；如果后续要支持未登录反馈，可把 create/options 拆到不需要鉴权的 group。
	api := feedback.Group("/api", requireAppAuth(d))
	{
		// 统一使用 POST
		api.POST("/create", d.FeedbackHandler.Create)

		api.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "feedback pong"})
		})
	}
}
