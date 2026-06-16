package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	adminctx "storeready_ai/internal/admin/auth/context"
	authdto "storeready_ai/internal/admin/modules/auth/dto"
	authservice "storeready_ai/internal/admin/modules/auth/service"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// Handler 是后台认证模块 handler。
//
// 说明：
// 1. 只承接 admin auth 模块相关 HTTP 请求；
// 2. handler 只做参数绑定、调用 service、输出响应，不直接操作 repo/model；
// 3. 响应统一走 response.AbortFail / WriteError / WriteOK。
type Handler struct {
	service authservice.Service
}

func New(service authservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	var req authdto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := getTenantIDFromRequest(c, req.TenantID)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) Login(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	var req authdto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	tenantId64, err := utils.ToUint64(req.TenantID)
	if err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), "invalid tenant_id", rid)
		return
	}
	if tenantId64 == 0 {
		req.TenantID = hander.GetTenantID(c)
	}

	loginIP := strings.TrimSpace(c.ClientIP())
	resp, err := h.service.Login(c.Request.Context(), req, loginIP)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	var req authdto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	resp, err := h.service.RefreshToken(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) Logout(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	var req authdto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if err := h.service.Logout(c.Request.Context(), req); err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, nil, rid)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	adminUserID, ok := getAdminUserID(c)
	if !ok || adminUserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "admin user not authenticated", rid)
		return
	}

	var req authdto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := getTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "tenant_id missing in context", rid)
		return
	}
	req.TenantID = tenantID

	if err := h.service.ChangePassword(c.Request.Context(), adminUserID, req); err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, nil, rid)
}

func (h *Handler) Me(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin auth service is nil", rid)
		return
	}

	adminUserID, ok := getAdminUserID(c)
	if !ok || adminUserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "admin user not authenticated", rid)
		return
	}

	tenantID, ok := getTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "tenant_id missing in context", rid)
		return
	}

	username, ok := adminctx.GetUsername(c)
	if !ok || username == "" {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "admin username missing in context", rid)
		return
	}

	isSuperAdmin := uint8(0)
	if adminctx.IsSuperAdmin(c) {
		isSuperAdmin = 1
	}

	roles := adminctx.GetRoles(c)
	resp := authdto.AdminUserProfile{
		TenantID:     tenantID,
		ID:           adminUserID,
		Username:     username,
		IsSuperAdmin: isSuperAdmin,
		Roles:        roles,
	}

	response.WriteOK(c, resp, rid)
}

func getTenantID(c *gin.Context) (uint64, bool) {
	if c == nil {
		return 0, false
	}
	return adminctx.GetTenantID(c)
}

func getTenantIDFromRequest(c *gin.Context, fallback uint64) (uint64, bool) {
	if fallback > 0 {
		return fallback, true
	}
	if c == nil {
		return 0, false
	}
	raw := strings.TrimSpace(c.Query("tenant_id"))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return v, true
}

func getAdminUserID(c *gin.Context) (uint64, bool) {
	if c == nil {
		return 0, false
	}
	return adminctx.GetAdminUserID(c)
}

func getRID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := strings.TrimSpace(c.GetString("rid")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-Id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	return ""
}
