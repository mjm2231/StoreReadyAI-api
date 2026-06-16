package handler

import (
	"fmt"
	"net/http"
	"storeready_ai/internal/admin/modules/appuser/dto"
	appuser "storeready_ai/internal/admin/modules/appuser/service"
	"storeready_ai/internal/common/rid"
	"storeready_ai/internal/contracts/user"
	"storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"

	adminctx "storeready_ai/internal/admin/auth/context"
	errx "storeready_ai/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc appuser.Service
}

func New(svc appuser.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListUsers(c *gin.Context) {
	rid := rid.GetRID(c)

	if c == nil {
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin appuser service is nil", rid)
		return
	}

	var req user.QueryUserFilter
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	resp := &dto.ListUsersResp{}
	users, total, err := h.svc.ListUsers(c.Request.Context(), tenantID, req)
	resp.Users = users
	resp.Total = total
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	rid := rid.GetRID(c)

	if c == nil {
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin appuser service is nil", rid)
		return
	}

	var req = &dto.GetUserByIDReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	uid, _ := utils.ToUint64(req.ID)
	u, err := h.svc.GetUserByID(c.Request.Context(), tenantID, uid)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}
	var resp = &dto.GetUserByIDResp{User: u}

	response.WriteOK(c, resp, rid)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	rid := rid.GetRID(c)

	if c == nil {
		return
	}
	if h == nil || h.svc == nil {
		response.AbortFail(c, http.StatusInternalServerError, int32(errx.CodeInternal), "admin appuser service is nil", rid)
		return
	}

	var req = &dto.UpdateUserReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortFail(c, http.StatusBadRequest, int32(errx.CodeInvalidParam), err.Error(), rid)
		return
	}

	tenantID, ok := adminctx.GetTenantID(c)
	if !ok || tenantID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeUnauthorized), "missing tenant_id", rid)
		return
	}
	fmt.Printf("UpdateUser %+v\n", req)
	err := h.svc.UpdateUser(c.Request.Context(), tenantID, req.ID, req.UpdateUserReq)
	if err != nil {
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, nil, rid)
}
