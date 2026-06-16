package handler

import (
	"net/http"
	"strings"

	appuserhandler "storeready_ai/internal/admin/modules/appuser/handler"
	audithandler "storeready_ai/internal/admin/modules/audit/handler"
	authhandler "storeready_ai/internal/admin/modules/auth/handler"
	feedbackhandler "storeready_ai/internal/admin/modules/feedback/handler"
	mehandler "storeready_ai/internal/admin/modules/me/handler"
	permissionhandler "storeready_ai/internal/admin/modules/permissions/handler"

	rolehandler "storeready_ai/internal/admin/modules/roles/handler"
	statshandler "storeready_ai/internal/admin/modules/stats/handler"

	userhandler "storeready_ai/internal/admin/modules/user/handler"

	"github.com/gin-gonic/gin"
)

// Handler 是 Admin 端通用 handler 聚合入口。
//
// 当前先提供最小骨架：
// 1. Health/Ping：用于健康检查；
// 2. Me：用于验证 admin jwt / middleware / context 注入链路；
// 3. 后续再按模块补充 user / role / permission / audit / auth 等 handler。
type Handler struct {

	// AuthHandler 后台认证模块 handler。
	// 当前用于承接 /admin-api/auth 下的认证接口。
	AuthHandler *authhandler.Handler

	// UserHandler 后台管理员用户模块 handler。
	// 当前用于承接 /admin-api/users 下的用户管理接口。
	UserHandler *userhandler.Handler

	// RoleHandler 后台角色模块 handler。
	// 当前用于承接 /admin-api/roles 下的角色管理接口。
	RoleHandler *rolehandler.Handler

	// PermissionHandler 后台权限模块 handler。
	// 当前用于承接 /admin-api/permissions 下的权限管理接口。
	PermissionHandler *permissionhandler.Handler

	// AuditHandler 后台审计模块 handler。
	// 当前用于承接 /admin-api/audit 下的审计接口。
	AuditHandler *audithandler.Handler

	// StatsHandler 后台统计模块 handler。
	// 当前用于承接 /admin-api/stats 下的统计接口。
	StatsHandler *statshandler.Handler
	// 权限管理 handler
	PermHandler *permissionhandler.Handler
	//角色管理 handler
	RolesHandler *rolehandler.Handler

	MeHandler *mehandler.Handler
	// AppUserHandler 后台应用用户模块 handler。
	AppUserHandler *appuserhandler.Handler
	//反馈模块
	FeedbackHandler *feedbackhandler.Handler
}

func (h *Handler) ok(c *gin.Context, msg string, data any) {
	if c == nil {
		return
	}
	if strings.TrimSpace(msg) == "" {
		msg = "OK"
	}
	resp := gin.H{
		"code": 0,
		"msg":  msg,
	}
	if data != nil {
		resp["data"] = data
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) okMessage(c *gin.Context, msg string) {
	h.ok(c, msg, nil)
}

func New() *Handler {
	return &Handler{}
}

// Ping 健康检查。
func (h *Handler) Ping(c *gin.Context) {
	h.okMessage(c, "pong")
}

// Me 返回当前已登录后台管理员的最小身份信息。
//
// 依赖：
// - admin middleware 已完成鉴权；
// - context 中已注入 admin_user_id / admin_username / admin_roles。
// func (h *Handler) Me(c *gin.Context) {
// 	if c == nil {
// 		return
// 	}

// 	adminUserID, _ := context.GetAdminUserID(c)
// 	username, _ := context.GetUsername(c)
// 	roles := context.GetRoles(c)
// 	if roles == nil {
// 		roles = make([]string, 0)
// 	}

// 	h.ok(c, "OK", gin.H{
// 		"admin_user_id": adminUserID,
// 		"username":      strings.TrimSpace(username),
// 		"roles":         roles,
// 	})
// }
