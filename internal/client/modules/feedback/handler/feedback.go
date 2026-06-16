package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/modules/feedback/dto"
	"storeready_ai/internal/client/modules/feedback/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/common/rid"
	"storeready_ai/internal/pkg/response"
)

// Handler 用户反馈接口处理器。
type Handler struct {
	service service.Service
}

// NewHandler 创建用户反馈接口处理器。
func NewHandler(service service.Service) *Handler {
	return &Handler{service: service}
}

// Create 创建用户反馈。
//
// @Summary 创建用户反馈
// @Description 客户端提交问题反馈、功能建议、支付订阅问题、账号登录问题等。未登录用户 uid 可为 0，服务端会记录客户端版本、平台、设备、系统版本和语言等环境信息。
// @Tags Feedback
// @Accept json
// @Produce json
// @Param body body dto.CreateFeedbackReq true "创建用户反馈请求"
// @Success 200 {object} response.DocResponse{data=dto.FeedbackVO} "成功返回反馈详情"
// @Failure 400 {object} response.DocError "参数错误或反馈内容为空"
// @Failure 500 {object} response.DocError "服务器内部错误"
// @Router /v1/feedback/api/create [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, 40000, err.Error(), rid.GetRID(c))
		return
	}

	tenantID := getTenantID(c)
	uid := getUID(c)
	meta := clientMetaFromContext(c)

	vo, err := h.service.Create(c.Request.Context(), tenantID, uid, req, meta)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.WriteOK(c, vo, rid.GetRID(c))
}

func clientMetaFromContext(c *gin.Context) service.ClientMeta {
	return service.ClientMeta{
		AppVersion:  strings.TrimSpace(c.GetHeader("X-App-Version")),
		BuildNumber: strings.TrimSpace(c.GetHeader("X-App-Build")),
		Platform:    firstNonEmpty(c.GetHeader("X-Platform"), c.GetHeader("X-Client-Platform")),
		DeviceModel: firstNonEmpty(c.GetHeader("X-Device-Model"), c.GetHeader("X-Device")),
		OSVersion:   strings.TrimSpace(c.GetHeader("X-OS-Version")),
		Locale:      firstNonEmpty(c.GetHeader("Accept-Language"), c.GetHeader("X-Locale")),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func getTenantID(c *gin.Context) uint64 {
	if tenantID, ok := common.GetTenantID(c); ok {
		return parseUint64(tenantID)
	}
	return parseUint64(c.GetHeader("X-Tenant-ID"))
}

func getUID(c *gin.Context) uint64 {
	if userID, ok := common.GetUserID(c); ok {
		return userID
	}
	if uid, ok := common.GetUID(c); ok {
		return parseUint64(uid)
	}
	return parseUint64(c.GetHeader("X-UID"))
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
