package handler

import (
	"net/http"
	"strconv"

	"storeready_ai/internal/client/middleware"
	"storeready_ai/internal/client/modules/project/dto"
	projectsvc "storeready_ai/internal/client/modules/project/service"
	"storeready_ai/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目相关 HTTP Handler。
//
// MVP 范围：
//  1. Ping：模块健康检查
//  2. Create：创建项目
//  3. List：当前登录用户的项目列表
//
// 注意：这里不做项目详情、编辑、删除、成员协作、后台项目管理。
type ProjectHandler struct {
	projectSvc projectsvc.ProjectService
}

func NewProjectHandler(projectSvc projectsvc.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectSvc: projectSvc}
}

func getRID(c *gin.Context) string {
	return middleware.GetRequestID(c)
}

// Ping 项目模块健康检查。
func (h *ProjectHandler) Ping(c *gin.Context) {
	response.WriteOK(c, gin.H{"msg": "project pong"}, getRID(c))
}

// Create 创建项目。
func (h *ProjectHandler) Create(c *gin.Context) {
	tenantID, _ := getUint64FromContext(c, "tenant_id", "tenantID", "TenantID")
	userID, ok := getUint64FromContext(c, "user_id", "userID", "UserID", "uid", "UID")
	if !ok || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, 401, "未登录或登录已失效", getRID(c))
		return
	}

	var req dto.CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 10001, err.Error(), getRID(c))
		return
	}

	resp, err := h.projectSvc.CreateProject(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.WriteError(c, err, getRID(c))
		return
	}

	response.WriteOK(c, resp, getRID(c))
}

// List 获取当前登录用户的项目列表。
func (h *ProjectHandler) List(c *gin.Context) {
	tenantID, _ := getUint64FromContext(c, "tenant_id", "tenantID", "TenantID")
	userID, ok := getUint64FromContext(c, "user_id", "userID", "UserID", "uid", "UID")
	if !ok || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, 401, "未登录或登录已失效", getRID(c))
		return
	}

	var req dto.ListProjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 10001, err.Error(), getRID(c))
		return
	}

	resp, err := h.projectSvc.ListProjects(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.WriteError(c, err, getRID(c))
		return
	}

	response.WriteOK(c, resp, getRID(c))
}

func getUint64FromContext(c *gin.Context, keys ...string) (uint64, bool) {
	for _, key := range keys {
		v, exists := c.Get(key)
		if !exists || v == nil {
			continue
		}

		switch val := v.(type) {
		case uint64:
			return val, true
		case uint:
			return uint64(val), true
		case int64:
			if val > 0 {
				return uint64(val), true
			}
		case int:
			if val > 0 {
				return uint64(val), true
			}
		case string:
			n, err := strconv.ParseUint(val, 10, 64)
			if err == nil {
				return n, true
			}
		}
	}
	return 0, false
}
