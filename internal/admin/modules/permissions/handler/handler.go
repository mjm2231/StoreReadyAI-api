package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	adminctx "storeready_ai/internal/admin/auth/context"
	permissiondto "storeready_ai/internal/admin/modules/permissions/dto"
	permissionservice "storeready_ai/internal/admin/modules/permissions/service"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
)

// Handler 是后台权限模块 handler。
//
// 说明：
// 1. 只承接 admin permissions 模块相关 HTTP 请求；
// 2. handler 只做参数绑定、调用 service、输出响应，不直接操作 repo/model；
// 3. 响应统一走 response.AbortFail / WriteError / WriteOK。
type Handler struct {
	service permissionservice.Service
}

func New(service permissionservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.PermissionListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	resp, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) Detail(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.GetPermissionDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	resp, err := h.service.GetDetail(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) Create(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	resp, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) Update(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	resp, err := h.service.Update(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.UpdatePermissionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	if err := h.service.UpdateStatus(c.Request.Context(), req); err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, nil, rid)
}

func (h *Handler) Delete(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin permission service is nil", rid)
		return
	}

	var req permissiondto.DeletePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	req.TenantID = tenantID

	if err := h.service.Delete(c.Request.Context(), req); err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, nil, rid)
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
