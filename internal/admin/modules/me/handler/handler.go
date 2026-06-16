package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"storeready_ai/internal/admin/auth/context"
	meservice "storeready_ai/internal/admin/modules/me/service"
	errx "storeready_ai/internal/pkg/errors"
	"storeready_ai/internal/pkg/response"
	utils "storeready_ai/internal/pkg/uitls"
)

type Handler struct {
	svc meservice.Service
}

func New(svc meservice.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Me(c *gin.Context) {
	rid := getRID(c)
	tenantID, ok := context.GetTenantID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeInternal), "missing tenant_id", rid)
		return
	}

	adminUserID, ok := context.GetAdminUserID(c)
	if !ok {
		response.AbortFail(c, http.StatusUnauthorized, int32(errx.CodeInternal), "missing admin user id", rid)
		return
	}

	slog.Info("admin me request",
		"tenant_id", tenantID,
		"admin_user_id", adminUserID,
	)
	tenantID64, _ := utils.ToUint64(tenantID)
	data, err := h.svc.GetCurrent(c.Request.Context(), tenantID64, adminUserID)
	if err != nil {
		slog.Error("admin me failed",
			"tenant_id", tenantID,
			"admin_user_id", adminUserID,
			"err", err,
		)
		response.WriteError(c, err, rid)
		return
	}

	response.WriteOK(c, data, rid)
}

func getRID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := strings.TrimSpace(c.GetString("rid")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-Id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	return ""
}
