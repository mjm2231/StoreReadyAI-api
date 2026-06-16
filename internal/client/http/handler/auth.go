package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/client/middleware"
	"storeready_ai/internal/client/modules/user/dto"
	usersvc "storeready_ai/internal/client/modules/user/service"
	"storeready_ai/internal/infra/hander"
	errx "storeready_ai/internal/pkg/errors"
	response "storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

// AuthHandler 认证相关 HTTP Handler。
type AuthHandler struct {
	userSvc usersvc.UserService
}

func NewAuthHandler(userSvc usersvc.UserService) *AuthHandler {
	return &AuthHandler{userSvc: userSvc}
}

func getRID(c *gin.Context) string {
	return middleware.GetRequestID(c)
}

// FirebaseLogin Firebase 第三方登录。
//
// @Summary Firebase 第三方登录
// @Description 使用 Firebase id_token 完成登录/注册，返回 token 与用户信息。
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body dto.FirebaseLoginReq true "Firebase 登录参数"
// @Success 200 {object} response.DocResponse{data=dto.FirebaseLoginResp} "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/auth/firebase/login [post]
func (h *AuthHandler) FirebaseLogin(c *gin.Context) {
	rid := getRID(c)

	var req dto.FirebaseLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if h == nil || h.userSvc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "user service not configured", rid)
		return
	}

	// MVP：单租户，tenant_id 固定为 0；后续多租户可从 JWT/域名/header 解析。
	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}

	resp, err := h.userSvc.FirebaseLogin(c.Request.Context(), tenantID, req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

// AccountRegister 账号密码注册。
//
// @Summary 账号密码注册
// @Description 使用邮箱和密码注册账号，注册成功后返回 token、用户信息与权益信息。
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body dto.AccountRegisterReq true "账号注册参数"
// @Success 200 {object} response.DocResponse{data=dto.AccountRegisterResp} "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/auth/account/register [post]
func (h *AuthHandler) AccountRegister(c *gin.Context) {
	rid := getRID(c)

	var req dto.AccountRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if h == nil || h.userSvc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "user service not configured", rid)
		return
	}

	// MVP：单租户，tenant_id 固定为 0；后续多租户可从 JWT/域名/header 解析。
	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}

	resp, err := h.userSvc.AccountRegister(c.Request.Context(), tenantID, req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

// AccountLogin 账号密码登录。
//
// @Summary 账号密码登录
// @Description 使用邮箱和密码登录，返回 token、用户信息与权益信息。
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body dto.AccountLoginReq true "账号登录参数"
// @Success 200 {object} response.DocResponse{data=dto.AccountLoginResp} "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/auth/account/login [post]
func (h *AuthHandler) AccountLogin(c *gin.Context) {
	rid := getRID(c)

	var req dto.AccountLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	if h == nil || h.userSvc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "user service not configured", rid)
		return
	}

	// MVP：单租户，tenant_id 固定为 0；后续多租户可从 JWT/域名/header 解析。
	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}

	resp, err := h.userSvc.AccountLogin(c.Request.Context(), tenantID, req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

// RefreshToken 刷新访问令牌。
//
// @Summary 刷新访问令牌
// @Description 使用 refresh_token 换取新的 access_token，并返回 refresh_token 与剩余有效期。
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body dto.RefreshTokenReq true "刷新令牌参数"
// @Success 200 {object} response.DocResponse{data=dto.RefreshTokenResp} "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	rid := getRID(c)

	var req dto.RefreshTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.userSvc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "user service not configured", rid)
		return
	}
	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	resp, err := h.userSvc.RefreshAccessToken(c.Request.Context(), tenantID, req)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, resp, rid)
}

// Logout 退出登录。
//
// @Summary 退出登录
// @Description 吊销 refresh_token，使其不可再用于刷新 access_token。
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body dto.LogoutReq true "退出登录参数"
// @Success 200 {object} response.DocResponse{data=map[string]interface{}} "成功"
// @Failure 400 {object} response.DocError "参数错误"
// @Failure 401 {object} response.DocError "未授权"
// @Failure 500 {object} response.DocError "服务端错误"
// @Router /v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	rid := getRID(c)

	var req dto.LogoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}
	if h == nil || h.userSvc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "user service not configured", rid)
		return
	}
	tenantID, err := utils.ToUint64(hander.GetTenantID(c))
	if err != nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "invalid tenant_id in context", rid)
		return
	}
	if err := h.userSvc.Logout(c.Request.Context(), tenantID, req); err != nil {
		response.WriteError(c, err, rid)
		return
	}
	response.WriteOK(c, map[string]any{}, rid)
}
