package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	"storeready_ai/internal/client/modules/billing/dto"
	billingsvc "storeready_ai/internal/client/modules/billing/service"
	"storeready_ai/internal/common"
	"storeready_ai/internal/i18n"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
)

// BillingHandler Billing HTTP 处理器。
//
// 职责：
// 1. 接收客户端购买校验 / 恢复购买请求；
// 2. 查询当前 entitlement；
// 3. 查询 Billing 页面配置；
// 4. 将 gin 上下文中的 tenant_id / uid / user_id 收口后传给 service。
type BillingHandler struct {
	service billingsvc.Service
	i18n    i18n.Translator
}

// NewBillingHandler 创建 BillingHandler。
func NewBillingHandler(service billingsvc.Service, i18n i18n.Translator) *BillingHandler {
	return &BillingHandler{
		service: service,
		i18n:    i18n,
	}
}

// VerifyPurchase 购买校验。
//
// @Summary 购买校验
// @Description 校验当前登录用户提交的 Google Play / App Store 购买凭证，并刷新对应权益状态。
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.VerifyPurchaseReq true "购买校验参数"
// @Success 200 {object} response.DocResponse{data=dto.VerifyPurchaseResp} "成功（data 为校验结果与最新权益）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/billing/api/verify [post]
func (h *BillingHandler) VerifyPurchase(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "billing service not configured", rid)
		return
	}

	var req dto.VerifyPurchaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, userID, uid, ok := readCurrentUser(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "unauthorized", rid)
		return
	}

	resp, err := h.service.VerifyPurchase(c.Request.Context(), tenantID, userID, uid, req)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// RestorePurchase 恢复购买。
//
// @Summary 恢复购买
// @Description 恢复当前登录用户已有的购买记录，并刷新对应权益状态。
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.RestorePurchaseReq true "恢复购买参数"
// @Success 200 {object} response.DocResponse{data=dto.RestorePurchaseResp} "成功（data 为恢复结果与最新权益）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/billing/api/restore [post]
func (h *BillingHandler) RestorePurchase(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "billing service not configured", rid)
		return
	}

	var req dto.RestorePurchaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, userID, uid, ok := readCurrentUser(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "unauthorized", rid)
		return
	}

	resp, err := h.service.RestorePurchase(c.Request.Context(), tenantID, userID, uid, req)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

// GetEntitlement 查询当前权益。
//
// @Summary 查询当前权益
// @Description 查询当前登录用户的 Billing 权益状态。
// @Tags Billing
// @Accept json
// @Produce json
// @Success 200 {object} response.DocResponse{data=dto.EntitlementResp} "成功（data 为当前权益信息）"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/billing/api/entitlement [post]
func (h *BillingHandler) GetEntitlement(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "billing service not configured", rid)
		return
	}

	tenantID, uid, ok := readTenantAndUID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "unauthorized", rid)
		return
	}

	resp, err := h.service.GetEntitlement(c.Request.Context(), tenantID, uid)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

// GetConfig 查询 Billing 配置。
//
// @Summary 查询 Billing 配置
// @Description 查询 Billing 页面展示配置与当前平台可用商品列表。
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.BillingConfigReq true "Billing 配置查询参数"
// @Success 200 {object} response.DocResponse{data=dto.BillingConfigResp} "成功（data 为 Billing 配置）"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/billing/api/config [post]
func (h *BillingHandler) GetConfig(c *gin.Context) {
	rid := middleware.GetRequestID(c)
	if h == nil || h.service == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "billing service not configured", rid)
		return
	}

	var req dto.BillingConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	platform := strings.TrimSpace(strings.ToLower(req.Platform))
	if platform == "" {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), "platform required", rid)
		return
	}

	tenantID, _ := readTenantID(c)
	resp, err := h.service.GetConfig(c.Request.Context(), tenantID, platform)
	if err != nil {
		writeServiceError(c, err, rid)
		return
	}

	locale := hander.GetLocale(c)
	for i := range resp.Products {
		productCode := resp.Products[i].ProductCode
		fmt.Printf("billng handler GetLocale: %v ,ProductCode: %v\n", locale, productCode)
		if !h.i18n.HasKey(locale, productCode) {
			resp.Products[i].Title = h.i18n.T(i18n.LocaleENUS, productCode, nil)
			continue
		}

		title := h.i18n.T(locale, productCode, nil)
		fmt.Printf("billng handler title: %v\n", title)
		resp.Products[i].Title = title
	}
	response.WriteOK(c, resp, rid)
}

func readCurrentUser(c *gin.Context) (tenantID, userID, uid uint64, ok bool) {
	tenantID, ok = readTenantID(c)
	if !ok {
		return 0, 0, 0, false
	}
	uid, ok = readUID(c)
	if !ok {
		return 0, 0, 0, false
	}
	userID, ok = common.GetUserID(c)
	if !ok || userID == 0 {
		return 0, 0, 0, false
	}
	return tenantID, userID, uid, true
}

func readTenantAndUID(c *gin.Context) (tenantID, uid uint64, ok bool) {
	tenantID, ok = readTenantID(c)
	if !ok {
		return 0, 0, false
	}
	uid, ok = readUID(c)
	if !ok {
		return 0, 0, false
	}
	return tenantID, uid, true
}

func readTenantID(c *gin.Context) (uint64, bool) {
	tenantIDStr, ok := common.GetTenantID(c)
	if !ok || strings.TrimSpace(tenantIDStr) == "" {
		return 0, false
	}
	n, err := parseUint64(tenantIDStr)
	if err != nil || n == 0 {
		return 0, false
	}
	return n, true
}

func readUID(c *gin.Context) (uint64, bool) {
	uidStr, ok := common.GetUID(c)
	if !ok || strings.TrimSpace(uidStr) == "" {
		return 0, false
	}
	n, err := parseUint64(uidStr)
	if err != nil || n == 0 {
		return 0, false
	}
	return n, true
}

func parseUint64(v string) (uint64, error) {
	var n uint64
	for _, ch := range strings.TrimSpace(v) {
		if ch < '0' || ch > '9' {
			return 0, http.ErrNotSupported
		}
		n = n*10 + uint64(ch-'0')
	}
	if n == 0 {
		return 0, http.ErrNoCookie
	}
	return n, nil
}

func writeServiceError(c *gin.Context, err error, rid string) {
	if err == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "unknown error", rid)
		return
	}
	// 当前先维持简单映射，后续可统一走项目中的 errors.CodeOf / HTTPStatusOf。
	msg := err.Error()
	status := http.StatusInternalServerError
	code := int32(errx.CodeInternal)
	if strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "required") {
		status = http.StatusBadRequest
		code = int32(errx.CodeInvalidParam)
	}
	if strings.Contains(strings.ToLower(msg), "unauthorized") {
		status = http.StatusUnauthorized
		code = int32(errx.CodeUnauthorized)
	}
	response.AbortFail(c, status, code, msg, rid)
}
