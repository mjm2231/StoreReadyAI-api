package handler

import (
	"errors"
	"net/http"
	"storeready_ai/internal/admin/modules/feedback/dto"
	"storeready_ai/internal/admin/modules/feedback/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/common/rid"
	"strconv"
	"strings"

	"storeready_ai/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.Service
}

func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// Detail 获取反馈详情。
//
// @Summary 获取反馈详情
// @Description 后台根据反馈 ID 获取用户反馈详情，包含反馈内容、分类、状态、优先级、客户端环境、回复和处理信息。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Param body body dto.FeedbackDetailReq true "反馈详情请求"
// @Success 200 {object} response.DocResponse "成功返回反馈详情"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 404 {object} response.DocError "反馈不存在"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/detail [post]
func (h *Handler) Detail(c *gin.Context) {
	var req dto.FeedbackDetailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	vo, err := h.svc.Detail(c.Request.Context(), getTenantID(c), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, vo, rid.GetRID(c))
}

// List 查询反馈列表。
//
// @Summary 查询反馈列表
// @Description 后台分页查询用户反馈，支持按 uid、分类、状态、优先级、关键词和创建时间范围筛选，默认按优先级和创建时间倒序返回。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Param body body dto.FeedbackListReq true "反馈列表查询请求"
// @Success 200 {object} response.DocResponse "成功返回反馈分页列表"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/list [post]
func (h *Handler) List(c *gin.Context) {
	var req dto.FeedbackListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	resp, err := h.svc.List(c.Request.Context(), getTenantID(c), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, resp, rid.GetRID(c))
}

// UpdateStatus 更新反馈状态。
//
// @Summary 更新反馈状态
// @Description 后台更新用户反馈处理状态，可选更新优先级。状态：1待处理 2处理中 3已处理 4已关闭；优先级：1低 2普通 3高 4紧急。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Param body body dto.UpdateFeedbackStatusReq true "更新反馈状态请求"
// @Success 200 {object} response.DocResponse "成功返回 ok=true"
// @Failure 400 {object} response.DocError "参数错误、状态非法或优先级非法"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/update-status [post]
func (h *Handler) UpdateStatus(c *gin.Context) {
	var req dto.UpdateFeedbackStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), getTenantID(c), getAdminID(c), req); err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, gin.H{"ok": true}, rid.GetRID(c))
}

// Reply 回复反馈。
//
// @Summary 回复用户反馈
// @Description 后台回复用户反馈。若请求未传 status，服务端默认将反馈状态置为已处理。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Param body body dto.ReplyFeedbackReq true "回复反馈请求"
// @Success 200 {object} response.DocResponse "成功返回 ok=true"
// @Failure 400 {object} response.DocError "参数错误或状态非法"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/reply [post]
func (h *Handler) Reply(c *gin.Context) {
	var req dto.ReplyFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	if err := h.svc.Reply(c.Request.Context(), getTenantID(c), getAdminID(c), req); err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, gin.H{"ok": true}, rid.GetRID(c))
}

// Delete 删除反馈。
//
// @Summary 删除用户反馈
// @Description 后台软删除用户反馈，仅标记 deleted_at，不物理删除数据。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Param body body dto.DeleteFeedbackReq true "删除反馈请求"
// @Success 200 {object} response.DocResponse "成功返回 ok=true"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/delete [post]
func (h *Handler) Delete(c *gin.Context) {
	var req dto.DeleteFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), getTenantID(c), req); err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, gin.H{"ok": true}, rid.GetRID(c))
}

// Options 获取反馈枚举选项。
//
// @Summary 获取反馈枚举选项
// @Description 获取反馈分类、处理状态和优先级枚举选项，用于后台筛选、表单下拉和前端展示。客户端也可复用该接口获取反馈分类。
// @Tags AdminFeedback
// @Accept json
// @Produce json
// @Success 200 {object} response.DocResponse "成功返回 categories/statuses/priorities"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /admin-api/feedback/options [post]
// @Router /v1/feedback/api/options [post]
func (h *Handler) Options(c *gin.Context) {
	resp := h.svc.Options(c.Request.Context())
	response.WriteOK(c, resp, rid.GetRID(c))
}

func getTenantID(c *gin.Context) uint64 {
	if tenantID, ok := common.GetTenantID(c); ok {
		return parseUint64(tenantID)
	}
	return parseUint64(c.GetHeader("X-Tenant-ID"))
}

func parseUint64(value string) uint64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func getAdminID(c *gin.Context) uint64 {
	if userID, ok := common.GetUserID(c); ok {
		return userID
	}
	return parseUint64(c.GetHeader("X-Admin-ID"))
}

func writeServiceError(c *gin.Context, err error) {
	if err == nil {
		response.WriteOK(c, gin.H{"ok": true}, rid.GetRID(c))
		return
	}

	switch {
	case errors.Is(err, service.ErrFeedbackContentRequired):
		response.AbortFail(c, http.StatusBadRequest, 40001, "feedback content required", rid.GetRID(c))
	case errors.Is(err, service.ErrInvalidFeedbackStatus):
		response.AbortFail(c, http.StatusBadRequest, 40002, "invalid feedback status", rid.GetRID(c))
	case errors.Is(err, service.ErrInvalidFeedbackPriority):
		response.AbortFail(c, http.StatusBadRequest, 40003, "invalid feedback priority", rid.GetRID(c))
	case errors.Is(err, service.ErrFeedbackNotFound):
		response.AbortFail(c, http.StatusNotFound, 40401, "feedback not found", rid.GetRID(c))
	default:
		response.WriteError(c, err, rid.GetRID(c))
	}
}
