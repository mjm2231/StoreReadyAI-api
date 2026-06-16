package middleware

import (
	"net/http"
	"strings"

	authctx "storeready_ai/internal/admin/auth/context"

	"github.com/gin-gonic/gin"
)

// PermissionConfig 是后台权限中间件配置。
//
// 设计说明：
// 1. Auth 与 Permission 分层：Auth 只负责“你是谁”，Permission 只负责“你能做什么”；
// 2. 当前先基于角色做最小权限控制，后续可平滑扩展 permission codes / super admin / data scope；
// 3. 若路由未先经过 admin Auth 中间件，这里的角色检查不会放行。
type PermissionConfig struct {
	// AnyRoles 满足任意一个角色即可通过。
	AnyRoles []string

	// AllRoles 必须同时具备这些角色才可通过。
	AllRoles []string

	// SuperAdminRoles 视为超级管理员角色，命中后直接放行。
	// 若未配置，则默认使用 super_admin。
	SuperAdminRoles []string

	// 失败响应。
	RespCode    int16
	RespMessage string
}

func (c *PermissionConfig) withDefaults() PermissionConfig {
	cfg := *c
	cfg.AnyRoles = normalizeRoles(cfg.AnyRoles)
	cfg.AllRoles = normalizeRoles(cfg.AllRoles)
	cfg.SuperAdminRoles = normalizeRoles(cfg.SuperAdminRoles)
	if len(cfg.SuperAdminRoles) == 0 {
		cfg.SuperAdminRoles = []string{"super_admin"}
	}
	if cfg.RespCode == 0 {
		cfg.RespCode = 403
	}
	if strings.TrimSpace(cfg.RespMessage) == "" {
		cfg.RespMessage = "Forbidden"
	}
	return cfg
}

// RequirePermission 统一后台权限中间件。
//
// 当前规则：
// 1. 必须先有合法管理员身份（admin auth context 中存在 admin_user_id）；
// 2. 若命中超级管理员角色，直接放行；
// 3. 若配置了 AnyRoles，则满足任意一个即可；
// 4. 若配置了 AllRoles，则必须全部满足；
// 5. AnyRoles / AllRoles 都为空时，默认只要求“已登录管理员”。
func RequirePermission(cfg PermissionConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()

	return func(c *gin.Context) {
		if c == nil {
			return
		}

		adminUserID, ok := authctx.GetAdminUserID(c)
		if !ok || adminUserID == 0 {
			abortForbidden(c, cfg, "admin_identity_required")
			return
		}

		roles := authctx.GetRoles(c)
		if hasAnyRole(roles, cfg.SuperAdminRoles) {
			c.Next()
			return
		}

		if len(cfg.AnyRoles) == 0 && len(cfg.AllRoles) == 0 {
			c.Next()
			return
		}

		if len(cfg.AnyRoles) > 0 && !hasAnyRole(roles, cfg.AnyRoles) {
			abortForbidden(c, cfg, "missing_required_role")
			return
		}

		if len(cfg.AllRoles) > 0 && !hasAllRoles(roles, cfg.AllRoles) {
			abortForbidden(c, cfg, "missing_required_role")
			return
		}

		c.Next()
	}
}

// RequireAnyRole 要求命中任意一个角色。
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return RequirePermission(PermissionConfig{AnyRoles: roles})
}

// RequireAllRoles 要求同时具备所有角色。
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return RequirePermission(PermissionConfig{AllRoles: roles})
}

// RequireSuperAdmin 要求具备超级管理员角色。
func RequireSuperAdmin() gin.HandlerFunc {
	return RequirePermission(PermissionConfig{AnyRoles: []string{"super_admin"}})
}

func abortForbidden(c *gin.Context, cfg PermissionConfig, code string) {
	if !c.Writer.Written() {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code": code,
			"msg":  cfg.RespMessage,
		})
		return
	}
	c.Abort()
}

func normalizeRoles(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func hasAnyRole(current []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(current) == 0 {
		return false
	}
	currentSet := make(map[string]struct{}, len(current))
	for _, item := range current {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		currentSet[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := currentSet[item]; ok {
			return true
		}
	}
	return false
}

func hasAllRoles(current []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(current) == 0 {
		return false
	}
	currentSet := make(map[string]struct{}, len(current))
	for _, item := range current {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		currentSet[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := currentSet[item]; !ok {
			return false
		}
	}
	return true
}
