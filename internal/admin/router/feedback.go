package router

import (
	"net/http"

	adminmw "storeready_ai/internal/admin/middleware"
	feedbackhandler "storeready_ai/internal/admin/modules/feedback/handler"

	"github.com/gin-gonic/gin"
)

func registerFeedbackRoutes(r gin.IRouter, d Deps) {
	if r == nil {
		return
	}

	h := d.Handler.FeedbackHandler
	if h == nil {
		return
	}
	adminFeedback := r.Group("/feedback", adminmw.RequireAnyRole("super_admin", "admin"))
	{
		// 统一使用 POST
		adminFeedback.POST("/list", h.List)
		adminFeedback.POST("/detail", h.Detail)
		adminFeedback.POST("/options", h.Options)
		adminFeedback.POST("/update-status", h.UpdateStatus)
		adminFeedback.POST("/reply", h.Reply)
		adminFeedback.POST("/delete", h.Delete)

		adminFeedback.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "feedback pong"})
		})
	}
}

var _ *feedbackhandler.Handler
