package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/admin/modules/stats/dto"
	statsservice "storeready_ai/internal/admin/modules/stats/service"
	"storeready_ai/internal/common/rid"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// Handler 是后台统计模块 handler。
//
// 说明：
// 1. 只承接 admin stats 模块相关 HTTP 请求；
// 2. handler 只做参数绑定、调用 service、输出响应，不直接操作 repo/model；
// 3. 响应统一走 response.AbortFail / WriteError / WriteOK。
type Handler struct {
	service *statsservice.Service
}

// New 创建后台统计 handler。
func New(service *statsservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOverview(c *gin.Context) {
	rid := rid.GetRID(c)

	if c == nil {
		return
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin stats service is nil", rid)
		return
	}

	var req dto.OverviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	tenantId, err := utils.ToUint64(req.TenantID)
	if err != nil {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}

	resp, err := h.service.GetOverview(c.Request.Context(), tenantId, time.Now())
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) getTenantIDOrAbort(c *gin.Context, tenantID string, ridValue string) (uint64, bool) {
	parsed, err := utils.ToUint64(tenantID)
	if err != nil {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", ridValue)
		return 0, false
	}
	return parsed, true
}

func (h *Handler) ensureServiceOrAbort(c *gin.Context, ridValue string) bool {
	if c == nil {
		return false
	}
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin stats service is nil", ridValue)
		return false
	}
	return true
}

func (h *Handler) GetSubscriptionCreatedTrend(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.TrendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionCreatedTrend(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetSubscriptionCycleStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionCycleStats(c.Request.Context(), tenantID)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetSubscriptionCurrencyStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.LimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionCurrencyStats(c.Request.Context(), tenantID, req.Limit)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetReminderSentTrend(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.TrendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetReminderSentTrend(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetReminderStatusStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetReminderStatusStats(c.Request.Context(), tenantID)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetUserCreatedTrend(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.TrendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetUserCreatedTrend(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetUserCoreStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetUserCoreStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetLoginStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetLoginStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetSubscriptionCoreStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionCoreStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetSubscriptionFormStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionFormStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetSubscriptionSyncStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetSubscriptionSyncStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetReminderCoreStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetReminderCoreStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetReminderSettingsStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetReminderSettingsStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetOverviewPageStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetOverviewPageStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

func (h *Handler) GetVipCoreStats(c *gin.Context) {
	ridValue := rid.GetRID(c)

	if !h.ensureServiceOrAbort(c, ridValue) {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), ridValue)
		return
	}
	req = req.Normalize()

	tenantID, ok := h.getTenantIDOrAbort(c, req.TenantID, ridValue)
	if !ok {
		return
	}

	resp, err := h.service.GetVipCoreStats(c.Request.Context(), tenantID, req.StartDate, req.EndDate)
	if err != nil {
		response.WriteError(c, err, ridValue)
		return
	}

	response.WriteOK(c, resp, ridValue)
}

var _ = (*dto.OverviewStats)(nil)
