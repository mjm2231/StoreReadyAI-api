package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterProjectRoutes registers project-related routes.
//
// MVP：项目模块只提供登录后的基础项目能力：
//  1. POST /v1/projects/ping：模块健康检查，不要求登录
//  2. POST /v1/projects/create：创建项目
//  3. POST /v1/projects/list：获取当前登录用户的项目列表
//  4. POST /v1/projects/detail：获取项目详情和上架资料
//  5. POST /v1/projects/store-info/save：保存项目上架资料
//  6. POST /v1/projects/store-info/generate：AI 生成上架资料文本
//
// 注意：
//  1. 项目约定业务接口统一使用 POST。
//  2. 当前不做项目编辑、删除、成员协作、发布状态流。
//  3. AI 生成只返回结果，不自动保存。
//  4. 这里采用显式接线，由 router.go 调用，不使用 init 自注册。
func RegisterProjectRoutes(r *gin.Engine, d Deps) {
	v1 := r.Group("/v1")

	// Public health for this module.
	v1.POST("/projects/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "project pong"})
	})

	if d.ProjectHandler == nil {
		panic("router: ProjectHandler is nil (did you wire it in app layer?)")
	}

	// Login required.
	projects := v1.Group("/projects", requireAppAuth(d))
	{
		projects.POST("/create", d.ProjectHandler.Create)
		projects.POST("/list", d.ProjectHandler.List)
		projects.POST("/detail", d.ProjectHandler.Detail)
		projects.POST("/store-info/save", d.ProjectHandler.SaveStoreInfo)
		projects.POST("/store-info/generate", d.ProjectHandler.GenerateStoreInfo)
	}
}
