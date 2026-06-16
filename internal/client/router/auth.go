package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers auth-related routes.
// MVP: Firebase third-party login + web account/password auth.
// 注意：这里采用显式接线（由 router.go 调用），不使用 init 自注册。
func RegisterAuthRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")
	auth := v1.Group("/auth")

	// Health for this module
	auth.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "auth pong"})
	})

	if d.AuthHandler == nil {
		panic("router: AuthHandler is nil (did you wire it in app layer?)")
	}
	auth.POST("/firebase/login", d.AuthHandler.FirebaseLogin)

	// Web 账号密码登录。
	auth.POST("/account/register", d.AuthHandler.AccountRegister)
	auth.POST("/account/login", d.AuthHandler.AccountLogin)

	// Token lifecycle.
	auth.POST("/refresh", d.AuthHandler.RefreshToken)
	auth.POST("/logout", d.AuthHandler.Logout)
	api := auth.Group("/api", requireAppAuth(d))
	{
		api.POST("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "auth pong"})
		})
	}
}
