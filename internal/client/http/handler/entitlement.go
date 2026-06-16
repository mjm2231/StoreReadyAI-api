package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	entdto "storeready_ai/internal/client/modules/entitlement/dto"
	entsvc "storeready_ai/internal/client/modules/entitlement/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	response "storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// EntitlementHandler 权益（VIP）相关 HTTP Handler。
//
// 说明：
// - 统一使用 POST。
// - 用户身份统一通过 common/ctx.go 的 GetUID 获取。
// - MVP：单租户 tenant_id 固定为 0。
//
// 路由建议：
// - POST /v1/vip/api/status
// - POST /v1/vip/api/grant   （仅后台/调试）
// - POST /v1/vip/api/revoke  （仅后台/调试）
//
// 注意：
// - grant/revoke 建议只给 admin/内网使用；客户端仅调用 status。
type EntitlementHandler struct {
	svc *entsvc.Service
}

// NewEntitlementHandler 创建 EntitlementHandler。
func NewEntitlementHandler(svc *entsvc.Service) *EntitlementHandler {
	return &EntitlementHandler{svc: svc}
}

// Status 查询 VIP 状态。
//
// @Summary 查询 VIP 状态
// @Description 查询当前登录用户的 VIP 状态与到期时间等信息。
// @Tags 权益
// @Accept json
// @Produce json
// @Param body body dto.GetVIPStatusReq false "查询参数（可选）"
// @Success 200 {object} response.DocResponse{data=dto.VIPStatusResp} "成功（data 为 VIP 状态）"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/vip/api/status [post]
func (h *EntitlementHandler) Status(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req entdto.GetVIPStatusReq
	_ = c.ShouldBindJSON(&req)

	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "entitlement service not configured", rid)
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

	resp, err := h.svc.GetVIPStatus(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Grant 手动开通/延长 VIP（仅后台/调试）。
//
// @Summary 手动开通/延长 VIP
// @Description 手动为当前登录用户开通或延长 VIP（建议仅后台/内网/调试使用）。
// @Tags 权益
// @Accept json
// @Produce json
// @Param body body dto.GrantVIPReq true "开通/延长参数"
// @Success 200 {object} response.DocResponse{dto.VIPStatusResp} "成功（data 为变更结果）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/vip/api/grant [post]
func (h *EntitlementHandler) Grant(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req entdto.GrantVIPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "entitlement service not configured", rid)
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

	resp, err := h.svc.GrantVIP(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Revoke 撤销 VIP（仅后台/调试）。
//
// @Summary 撤销 VIP
// @Description 撤销当前登录用户的 VIP 权益（建议仅后台/内网/调试使用）。
// @Tags 权益
// @Accept json
// @Produce json
// @Param body body dto.RevokeVIPReq false "撤销参数（可选）"
// @Success 200 {object} response.DocResponse{data=dto.VIPStatusResp} "成功（data 为变更结果）"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/vip/api/revoke [post]
func (h *EntitlementHandler) Revoke(c *gin.Context) {
	rid := middleware.GetRequestID(c)

	var req entdto.RevokeVIPReq
	_ = c.ShouldBindJSON(&req)
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "entitlement service not configured", rid)
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

	resp, err := h.svc.RevokeVIP(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}
