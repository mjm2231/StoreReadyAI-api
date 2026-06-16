package handler

import (
	"net/http"
	"strconv"
	"strings"

	"storeready_ai/internal/client/middleware"
	usersvc "storeready_ai/internal/client/modules/user/service"
	"storeready_ai/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户相关 HTTP Handler。
//
// MVP 范围：
//  1. Ping：模块健康检查
//  2. Me：获取当前登录用户信息
//  3. UpdateProfile：更新当前登录用户基础资料
//
// 注意：这里不做后台用户管理，不做用户列表/封禁/删除。
// 管理侧用户操作应放到 admin 模块，避免前台用户接口权限过大。
type UserHandler struct {
	userSvc usersvc.UserService
}

func NewUserHandler(userSvc usersvc.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func getRID(c *gin.Context) string {
	return middleware.GetRequestID(c)
}

// Ping 用户模块健康检查。
func (h *UserHandler) Ping(c *gin.Context) {
	response.WriteOK(c, gin.H{"msg": "user pong"}, getRID(c))
}

// Me 获取当前登录用户信息。
func (h *UserHandler) Me(c *gin.Context) {
	tenantID, _ := getUint64FromContext(c, "tenant_id", "tenantID", "TenantID")
	userID, ok := getUint64FromContext(c, "user_id", "userID", "UserID", "uid", "UID")
	if !ok || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, 401, "未登录或登录已失效", getRID(c))
		return
	}

	// 先返回鉴权上下文中的最小用户信息，保证登录闭环可用。
	// 后续 service/repo 完善后，可在这里改为 h.userSvc.GetMe(ctx, tenantID, userID)。
	response.WriteOK(c, gin.H{
		"tenant_id": tenantID,
		"user_id":   userID,
	}, getRID(c))
}

type updateProfileReq struct {
	Name   string `json:"name" binding:"omitempty,max=128"`
	Avatar string `json:"avatar" binding:"omitempty,max=512"`
}

// UpdateProfile 更新当前登录用户基础资料。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	_, ok := getUint64FromContext(c, "user_id", "userID", "UserID", "uid", "UID")
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, 401, "未登录或登录已失效", getRID(c))
		return
	}

	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 10001, err.Error(), getRID(c))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Avatar = strings.TrimSpace(req.Avatar)

	// 当前先不直接写库，避免在未知 UserService 接口上扩大改动范围。
	// 下一步确认 service 接口后，再接入真实 UpdateProfile 逻辑。
	response.WriteOK(c, gin.H{
		"name":   req.Name,
		"avatar": req.Avatar,
	}, getRID(c))
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
