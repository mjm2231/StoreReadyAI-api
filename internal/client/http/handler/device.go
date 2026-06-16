package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	devicedto "storeready_ai/internal/client/modules/device/dto"
	devicesvc "storeready_ai/internal/client/modules/device/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	response "storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// DeviceHandler 设备相关 HTTP Handler。
//
// 说明：
// - 统一使用 POST。
// - 用户身份统一通过 common/ctx.go 的 GetUID 获取。
// - MVP：单租户 tenant_id 固定为 0。
//
// 路由建议：
// - POST /v1/devices/api/register
// - POST /v1/devices/api/heartbeat
// - POST /v1/devices/api/touch_sync
// - POST /v1/devices/api/list
// - POST /v1/devices/api/revoke
//
// 注意：
// - last_ip/user_agent 由服务端从请求获取，不建议客户端传。
type DeviceHandler struct {
	svc *devicesvc.Service
}

// NewDeviceHandler 创建 DeviceHandler。
func NewDeviceHandler(svc *devicesvc.Service) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

// Register 设备登记。
//
// @Summary 设备登记
// @Description 绑定/更新当前登录用户的设备信息。服务端会记录 last_ip / user_agent，不建议客户端传。
// @Tags 设备
// @Accept json
// @Produce json
// @Param body body dto.RegisterDeviceReq true "设备登记参数"
// @Success 200 {object} response.DocResponse{data=dto.DeviceItem} "成功（data 为登记结果）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/devices/api/register [post]
func (h *DeviceHandler) Register(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req devicedto.RegisterDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "device service not configured", rid)
		return
	}

	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	uidStr, ok := common.GetUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}
	userID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}

	ip := clientIP(c)
	ua := clientUA(c)

	resp, err := h.svc.Register(c.Request.Context(), tenantID, userID, &req, ip, ua)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Heartbeat 设备心跳/活跃上报。
//
// @Summary 设备心跳
// @Description 上报设备活跃信息（用于在线状态/最后活跃时间）。
// @Tags 设备
// @Accept json
// @Produce json
// @Param body body dto.HeartbeatReq true "心跳参数"
// @Success 200 {object} response.DocResponse "成功（data 为空对象）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/devices/api/heartbeat [post]
func (h *DeviceHandler) Heartbeat(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req devicedto.HeartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "device service not configured", rid)
		return
	}

	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	uidStr, ok := common.GetUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}
	userID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}

	ip := clientIP(c)
	ua := clientUA(c)

	if err := h.svc.Heartbeat(c.Request.Context(), tenantID, userID, &req, ip, ua); err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, map[string]any{}, rid)
}

// TouchSync 同步时间点上报。
//
// @Summary 同步时间点上报
// @Description 上报客户端触发同步的时间点（用于增量同步/同步状态统计）。
// @Tags 设备
// @Accept json
// @Produce json
// @Param body body dto.TouchSyncReq true "同步触发参数"
// @Success 200 {object} response.DocResponse "成功（data 为空对象）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/devices/api/touch_sync [post]
func (h *DeviceHandler) TouchSync(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req devicedto.TouchSyncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "device service not configured", rid)
		return
	}

	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	uidStr, ok := common.GetUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}
	userID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}

	if err := h.svc.TouchSync(c.Request.Context(), tenantID, userID, &req); err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, map[string]any{}, rid)
}

// List 设备列表（仅 active）。
//
// @Summary 设备列表
// @Description 获取当前登录用户的设备列表（仅返回 active 设备）。
// @Tags 设备
// @Accept json
// @Produce json
// @Param body body dto.ListDevicesReq false "列表筛选参数（可选）"
// @Success 200 {object} response.DocResponse{data=dto.ListDevicesResp} "成功（data 为设备列表）"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/devices/api/list [post]
func (h *DeviceHandler) List(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req devicedto.ListDevicesReq
	_ = c.ShouldBindJSON(&req)
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "device service not configured", rid)
		return
	}

	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	uidStr, ok := common.GetUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}
	userID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}

	resp, err := h.svc.ListActiveDevices(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Revoke 撤销设备（踢设备）。
//
// @Summary 撤销设备
// @Description 撤销指定设备，使其失效（可用于踢出其它设备）。
// @Tags 设备
// @Accept json
// @Produce json
// @Param body body dto.RevokeDeviceReq true "撤销设备参数"
// @Success 200 {object} response.DocResponse "成功（data 为空对象）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/devices/api/revoke [post]
func (h *DeviceHandler) Revoke(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req devicedto.RevokeDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "device service not configured", rid)
		return
	}

	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	uidStr, ok := common.GetUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}
	userID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || userID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return
	}

	if err := h.svc.Revoke(c.Request.Context(), tenantID, userID, &req); err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, map[string]any{}, rid)
}

func clientIP(c *gin.Context) *string {
	if c == nil {
		return nil
	}
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return nil
	}
	return &ip
}

func clientUA(c *gin.Context) *string {
	if c == nil || c.Request == nil {
		return nil
	}
	ua := strings.TrimSpace(c.Request.UserAgent())
	if ua == "" {
		return nil
	}
	return &ua
}
