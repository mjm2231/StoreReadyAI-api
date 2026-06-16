package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	clienteventdto "storeready_ai/internal/client/modules/client_event/dto"
	clienteventsvc "storeready_ai/internal/client/modules/client_event/service"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
)

// ClientEventHandler 客户端埋点处理器。
//
// 职责：
// 1. 提供客户端埋点单条/批量上报入口；
// 2. 提供客户端埋点查询入口；
// 3. 统一从上下文读取 tenant_id / uid；
// 4. 统一输出项目标准响应结构。
type ClientEventHandler struct {
	service clienteventsvc.Service
}

// NewClientEventHandler 创建客户端埋点处理器。
func NewClientEventHandler(service clienteventsvc.Service) *ClientEventHandler {
	return &ClientEventHandler{service: service}
}

// Report 上报客户端埋点事件。
//
// @Summary 上报客户端埋点事件
// @Description 上报一条客户端埋点事件，用于 billing/sync/login/app 等链路排查。
// @Tags 客户端埋点
// @Accept json
// @Produce json
// @Param body body dto.ReportClientEventReq true "客户端埋点上报参数"
// @Success 200 {object} response.DocResponse "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/client-events/api/report [post]
func (h *ClientEventHandler) Report(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "client event service not configured", rid)
		return
	}

	var req clienteventdto.ReportClientEventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	log.Printf("client event report handler rid=%s event_id=%s event_group=%s event_name=%s", rid, req.EventID, req.EventGroup, req.EventName)

	tenantID, _ := readTenantID(c)
	uid, _ := readUID(c)
	result, err := h.service.Report(c.Request.Context(), tenantID, uid, req)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	requestCount := 0
	validCount := 0
	var insertedCount int64
	var duplicateCount int64
	if result != nil {
		requestCount = result.RequestCount
		validCount = result.ValidCount
		insertedCount = result.InsertedCount
		duplicateCount = result.DuplicateCount
	}

	response.WriteOK(c, gin.H{
		"reported":        true,
		"request_count":   requestCount,
		"valid_count":     validCount,
		"inserted_count":  insertedCount,
		"duplicate_count": duplicateCount,
	}, rid)
}

// ReportBatch 批量上报客户端埋点事件。
//
// @Summary 批量上报客户端埋点事件
// @Description 批量上报多条客户端埋点事件，用于客户端离线缓存后的集中补传。
// @Tags 客户端埋点
// @Accept json
// @Produce json
// @Param body body dto.ReportClientEventsBatchReq true "客户端埋点批量上报参数"
// @Success 200 {object} response.DocResponse "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/client-events/api/report-batch [post]
func (h *ClientEventHandler) ReportBatch(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "client event service not configured", rid)
		return
	}

	var req clienteventdto.ReportClientEventsBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	log.Printf("client event report batch handler rid=%s count=%d", rid, len(req.Items))
	for idx, item := range req.Items {
		log.Printf("client event report batch item rid=%s idx=%d event_id=%s event_group=%s event_name=%s", rid, idx, item.EventID, item.EventGroup, item.EventName)
	}

	tenantID, _ := readTenantID(c)
	uid, _ := readUID(c)
	result, err := h.service.ReportBatch(c.Request.Context(), tenantID, uid, req)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	requestCount := 0
	validCount := 0
	var insertedCount int64
	var duplicateCount int64
	if result != nil {
		requestCount = result.RequestCount
		validCount = result.ValidCount
		insertedCount = result.InsertedCount
		duplicateCount = result.DuplicateCount
	}

	response.WriteOK(c, gin.H{
		"reported":        true,
		"batch":           true,
		"count":           requestCount,
		"request_count":   requestCount,
		"valid_count":     validCount,
		"inserted_count":  insertedCount,
		"duplicate_count": duplicateCount,
	}, rid)
}

// List 查询客户端埋点事件列表。
//
// @Summary 查询客户端埋点事件列表
// @Description 按条件查询客户端埋点事件列表，用于后台排查客户端 billing/sync/login/app 等问题。
// @Tags 客户端埋点
// @Accept json
// @Produce json
// @Param body body dto.ListClientEventsReq true "客户端埋点查询参数"
// @Success 200 {object} response.DocResponse{data=dto.ListClientEventsResp} "成功（data 为埋点列表）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/client-events/api/list [post]
func (h *ClientEventHandler) List(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "client event service not configured", rid)
		return
	}

	var req clienteventdto.ListClientEventsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, _ := readTenantID(c)
	resp, err := h.service.List(c.Request.Context(), tenantID, req)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}
