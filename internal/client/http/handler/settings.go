package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	settingsdto "storeready_ai/internal/client/modules/settings/dto"
	settingssvc "storeready_ai/internal/client/modules/settings/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	response "storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// SettingsHandler 用户全局设置 HTTP Handler。
// 说明：
// - 统一使用 POST。
// - 用户身份统一通过 common/ctx.go 的 GetUID 获取。
// - tenant_id 必须来自认证上下文中的有效租户信息。
type SettingsHandler struct {
	svc *settingssvc.Service
}

// NewSettingsHandler 创建 SettingsHandler。
func NewSettingsHandler(svc *settingssvc.Service) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

func (h *SettingsHandler) parseActor(c *gin.Context) (tenantID uint64, userID uint64, rid string, ok bool) {
	rid = middleware.GetRequestID(c)

	parsedTenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil || parsedTenantID == 0 {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return 0, 0, rid, false
	}

	uidStr, exists := common.GetUID(c)
	if !exists {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return 0, 0, rid, false
	}

	parsedUserID, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || parsedUserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "未登录或用户信息缺失", rid)
		return 0, 0, rid, false
	}

	return parsedTenantID, parsedUserID, rid, true
}

// Get 获取当前用户设置（不存在则创建默认值）。
//
// @Summary 获取用户设置
// @Description 获取当前登录用户的全局设置；若不存在则创建默认值并返回。
// @Tags 设置
// @Accept json
// @Produce json
// @Param body body dto.GetSettingsReq false "查询参数（可选，允许空 body）"
// @Success 200 {object} response.DocResponse{data=dto.SettingsResp} "成功（data 为用户设置）"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/settings/api/get [post]
func (h *SettingsHandler) Get(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	// 允许空 body；若 body 非空且 JSON 非法，则返回 400。
	var req settingsdto.GetSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "settings service not configured", rid)
		return
	}

	tenantID, userID, rid, ok := h.parseActor(c)
	if !ok {
		return
	}

	resp, err := h.svc.Get(c.Request.Context(), tenantID, userID)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Update 更新当前用户设置（局部更新）。
//
// @Summary 更新用户设置
// @Description 局部更新当前登录用户的设置项（仅更新请求中提供的字段）。
// @Tags 设置
// @Accept json
// @Produce json
// @Param body body dto.UpdateSettingsReq true "更新参数"
// @Success 200 {object} response.DocResponse{data=dto.SettingsResp} "成功（data 为更新后的用户设置）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/settings/api/update [post]
func (h *SettingsHandler) Update(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req settingsdto.UpdateSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "settings service not configured", rid)
		return
	}

	tenantID, userID, rid, ok := h.parseActor(c)
	if !ok {
		return
	}

	resp, err := h.svc.Update(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}
