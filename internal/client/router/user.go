package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers user-related routes.
//
// MVP：用户模块只提供登录后的基础用户能力：
//  1. POST /v1/user/ping：模块健康检查，不要求登录
//  2. POST /v1/user/api/ping：登录态模块健康检查
//  3. POST /v1/user/api/me：获取当前登录用户
//  4. POST /v1/user/api/profile：更新当前登录用户资料
//
// 注意：
//  1. 项目约定业务接口统一使用 POST。
//  2. 后台用户管理不要放在这里，应放到 admin 模块。
//  3. 这里采用显式接线，由 router.go 调用，不使用 init 自注册。
func RegisterUserRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")

	// Public health for this module.
	v1.POST("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "user pong"})
	})

	if d.UserHandler == nil {
		panic("router: UserHandler is nil (did you wire it in app layer?)")
	}

	// Login required.
	user := v1.Group("/user", requireAppAuth(d))
	{
		user.GET("/me", d.UserHandler.Me)
		user.GET("/profile", d.UserHandler.UpdateProfile)
	}
}
