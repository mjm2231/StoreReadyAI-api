package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	auditdto "storeready_ai/internal/admin/modules/audit/dto"
	auditservice "storeready_ai/internal/admin/modules/audit/service"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
)

// Handler 是后台审计模块 handler。
//
// 说明：
// 1. 只承接 admin audit 模块相关 HTTP 请求；
// 2. handler 只做参数绑定、调用 service、输出响应，不直接操作 repo/model；
// 3. 响应统一走 response.AbortFail / WriteError / WriteOK。
type Handler struct {
	service auditservice.Service
}

func New(service auditservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin audit service is nil", rid)
		return
	}

	var req auditdto.CreateAuditLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	resp, err := h.service.Create(c.Request.Context(), req)
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
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin audit service is nil", rid)
		return
	}

	var req auditdto.GetAuditLogDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	resp, err := h.service.GetDetail(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) List(c *gin.Context) {
	rid := getRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin audit service is nil", rid)
		return
	}

	var req auditdto.AuditLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	resp, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
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
