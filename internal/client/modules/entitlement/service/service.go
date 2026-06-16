package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contractent "storeready_ai/internal/contracts/entitlement"

	entdto "storeready_ai/internal/client/modules/entitlement/dto"
	"storeready_ai/internal/client/modules/entitlement/model"
	entrepo "storeready_ai/internal/client/modules/entitlement/repo"
)

// Service 权益（VIP）服务（用例层）。
//
// 职责（MVP）：
// - 查询当前 VIP 状态（is_vip/expired_at/source 等）
// - （可选）手动开通/撤销（仅后台/调试）
//
// 说明：
// - tenant_id 在 MVP 固定为 0，但接口仍保留字段用于未来多租户扩展。
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - 本服务不处理 IAP 回调校验；后续可新增 verifier/handler 以 source=ios_iap/google_play 写入。
type Service struct {
	repo entrepo.EntitlementRepo
}

func New(r entrepo.EntitlementRepo) *Service {
	return &Service{repo: r}
}

// RefreshByBilling 根据 billing 订单结果刷新权益。
//
// 说明：
// 1. 当前 entitlement 模块继续作为权益真相；
// 2. billing 校验成功后，调用这里落一条 user_entitlements；
// 3. 目前先按 MVP 需求仅处理 VIP 权益。
func (s *Service) RefreshByBilling(ctx context.Context, req contractent.RefreshByBillingReq) (*contractent.CurrentEntitlement, error) {
	if req.TenantID == 0 || req.UserID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("entitlement service 未初始化")
	}

	now := uint64(time.Now().Unix())
	entitlement := model.EntitlementVIP
	if strings.TrimSpace(req.EntitlementCode) != "" {
		entitlement = req.EntitlementCode
	}

	productCode := strings.TrimSpace(req.ProductCode)
	productID := strings.TrimSpace(req.ProductID)
	if productCode == "" {
		productCode = productID
	}

	status := model.EntStatusActive
	switch strings.TrimSpace(strings.ToLower(req.Status)) {
	case string(model.EntStatusRevoked), "revoked":
		status = model.EntStatusRevoked
	case string(model.EntStatusExpired), "expired":
		status = model.EntStatusExpired
	case string(model.EntStatusActive), "active", "":
		status = model.EntStatusActive
	default:
		status = model.EntStatusActive
	}

	source := model.EntSourceManual
	switch strings.TrimSpace(strings.ToLower(req.Platform)) {
	case "android":
		source = model.EntSourceGooglePlay
	case "ios":
		source = model.EntSourceIOSIAP
	}

	autoRenew := model.AutoRenewOff
	if req.AutoRenew {
		autoRenew = model.AutoRenewOn
	}

	refID := strings.TrimSpace(req.PurchaseToken)
	if refID == "" {
		refID = strings.TrimSpace(req.OriginalOrderID)
	}
	var refPtr *string
	if refID != "" {
		refPtr = &refID
	}

	row := &model.UserEntitlement{
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Entitlement: entitlement,
		Source:      source,
		Status:      status,
		StartedAt:   now,
		ExpiredAt:   req.ExpiresAt,
		RefID:       refPtr,
		AutoRenew:   autoRenew,
		CreatedAt:   now,
		UpdatedAt:   now,
		ID:          0,
		ProductCode: productCode,
		ProductID:   productID,
	}

	var saved *model.UserEntitlement
	var err error
	if row.RefID != nil && strings.TrimSpace(*row.RefID) != "" {
		saved, err = s.repo.UpsertByRef(ctx, row)
	} else {
		saved, err = s.repo.Create(ctx, row)
	}
	if err != nil {
		return nil, err
	}
	fmt.Printf("RefreshByBilling Create UserEntitlement: %+v\n", saved)
	resp := toBillingEntitlementResp(saved, now)
	return resp, nil
}

// GetCurrent 查询当前用户权益。
//
// 说明：
//  1. 这里当前按 MVP 约定，将入参 uid 直接作为 user_id 使用；
//  2. 若后续 uid 与 users.id 分离，应在上层先完成 uid -> user_id 解析后再调用，
//     或在 entitlement service 中注入 resolver 统一转换。
func (s *Service) GetCurrent(ctx context.Context, tenantID, uid uint64, entitlementCode string) (*contractent.CurrentEntitlement, error) {
	if tenantID == 0 || uid == 0 {
		return nil, errors.New("tenant_id/uid 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("entitlement service 未初始化")
	}

	userID := uid
	now := uint64(time.Now().Unix())
	entitlement := model.EntitlementVIP
	if strings.TrimSpace(entitlementCode) != "" {
		entitlement = entitlementCode
	}

	row, err := s.repo.GetActive(ctx, tenantID, userID, entitlement, now)
	if err == nil && row != nil {
		return toBillingEntitlementResp(row, now), nil
	}

	latest, latestErr := s.repo.GetLatest(ctx, tenantID, userID, entitlement)
	if latestErr != nil {
		return toBillingEntitlementResp(nil, now), nil
	}
	return toBillingEntitlementResp(latest, now), nil
}

// GetVIPStatus 查询当前用户 VIP 状态。
// Route: POST /v1/vip/api/status
func (s *Service) GetVIPStatus(ctx context.Context, tenantID, userID uint64, _ *entdto.GetVIPStatusReq) (*entdto.VIPStatusResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("entitlement service 未初始化")
	}

	now := uint64(time.Now().Unix())
	row, err := s.repo.GetActive(ctx, tenantID, userID, model.EntitlementVIP, now)
	if err != nil {
		return nil, err
	}
	if row == nil {
		// 没有生效权益：尝试返回最新一条（用于展示 expired/revoked 状态）
		latest, err := s.repo.GetLatest(ctx, tenantID, userID, model.EntitlementVIP)
		if err != nil {
			// 没有任何记录时，返回默认未开通
			return &entdto.VIPStatusResp{
				Entitlement: model.EntitlementVIP,
				IsVIP:       false,
				Status:      model.EntStatusRevoked,
				Source:      model.EntSourceManual,
				AutoRenew:   model.AutoRenewOff,
				StartedAt:   0,
				ExpiredAt:   0,
				RefID:       nil,
				UpdatedAt:   0,
			}, nil
		}
		return toStatusResp(latest, now), nil
	}
	return toStatusResp(row, now), nil
}

// GrantVIP 手动开通/延长 VIP（仅后台/调试）。
// Route: POST /v1/vip/api/grant
func (s *Service) GrantVIP(ctx context.Context, tenantID, userID uint64, req *entdto.GrantVIPReq) (*entdto.VIPStatusResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("entitlement service 未初始化")
	}
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	now := uint64(time.Now().Unix())
	exp := uint64(0)
	if req.ExpiredAt != nil && *req.ExpiredAt > 0 {
		exp = *req.ExpiredAt
	} else {
		// 默认 30 天
		dur := uint64(30 * 24 * 3600)
		if req.DurationSeconds != nil && *req.DurationSeconds > 0 {
			dur = *req.DurationSeconds
		}
		exp = now + dur
	}

	source := model.EntSourceManual
	if req.Source != nil {
		source = *req.Source
	}
	autoRenew := model.AutoRenewOff
	if req.AutoRenew != nil {
		autoRenew = *req.AutoRenew
	}

	row := &model.UserEntitlement{
		TenantID:    tenantID,
		UserID:      userID,
		Entitlement: model.EntitlementVIP,
		Source:      source,
		Status:      model.EntStatusActive,
		StartedAt:   now,
		ExpiredAt:   exp,
		RefID:       trimPtr(req.RefID),
		AutoRenew:   autoRenew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 有 ref_id 则用幂等写入
	var saved *model.UserEntitlement
	var err error
	if row.RefID != nil && strings.TrimSpace(*row.RefID) != "" {
		saved, err = s.repo.UpsertByRef(ctx, row)
	} else {
		saved, err = s.repo.Create(ctx, row)
	}
	if err != nil {
		return nil, err
	}

	return toStatusResp(saved, now), nil
}

// RevokeVIP 撤销 VIP（仅后台/调试）。
// Route: POST /v1/vip/api/revoke
func (s *Service) RevokeVIP(ctx context.Context, tenantID, userID uint64, req *entdto.RevokeVIPReq) (*entdto.VIPStatusResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("entitlement service 未初始化")
	}
	_ = req

	now := uint64(time.Now().Unix())
	active, err := s.repo.GetActive(ctx, tenantID, userID, model.EntitlementVIP, now)
	if err != nil {
		return nil, err
	}
	if active == nil {
		// 没有生效记录，返回当前状态
		return s.GetVIPStatus(ctx, tenantID, userID, &entdto.GetVIPStatusReq{})
	}

	if err := s.repo.UpdateStatus(ctx, tenantID, userID, active.ID, model.EntStatusRevoked, now); err != nil {
		return nil, err
	}

	active.Status = model.EntStatusRevoked
	active.UpdatedAt = now
	return toStatusResp(active, now), nil
}

// --- helpers ---

func toBillingEntitlementResp(row *model.UserEntitlement, nowUnix uint64) *contractent.CurrentEntitlement {
	if row == nil {
		entitlement := contractent.Entitlement{
			IsVIP:                  0,
			VIPStartedAt:           0,
			VIPExpiredAt:           0,
			FreeLimit:              contractent.DefaultFreeLimit,
			SyncEnabled:            false,
			UnlimitedSubscriptions: false,
		}
		entitlement.Normalize()
		return &contractent.CurrentEntitlement{
			Entitlement:     entitlement,
			EntitlementCode: model.EntitlementVIP,
			Platform:        model.EntitlementSourceText(99),
			Status:          model.EntitlementStatusText(model.EntStatusExpired),
			AutoRenew:       false,
			ExpiresAt:       0,
			ProductCode:     "",
			ProductID:       "",
		}
	}

	var isVIP uint8
	if row.Status == model.EntStatusActive &&
		row.StartedAt > 0 &&
		row.StartedAt <= nowUnix &&
		row.ExpiredAt > 0 &&
		nowUnix < row.ExpiredAt {
		isVIP = 1
	}
	entitlement := contractent.Entitlement{
		IsVIP:                  isVIP,
		VIPStartedAt:           row.StartedAt,
		VIPExpiredAt:           row.ExpiredAt,
		FreeLimit:              contractent.DefaultFreeLimit,
		SyncEnabled:            isVIP == 1,
		UnlimitedSubscriptions: isVIP == 1,
	}
	entitlement.Normalize()
	return &contractent.CurrentEntitlement{
		Entitlement:     entitlement,
		EntitlementCode: model.EntitlementVIP,
		Platform:        model.EntitlementSourceText(row.Source),
		Status:          model.EntitlementStatusText(row.Status),
		AutoRenew:       model.EntitlementAutoRenew(row.AutoRenew),
		ExpiresAt:       row.ExpiredAt,
		ProductCode:     row.ProductCode,
		ProductID:       row.ProductID,
	}
}

func toStatusResp(row *model.UserEntitlement, nowUnix uint64) *entdto.VIPStatusResp {
	if row == nil {
		return &entdto.VIPStatusResp{Entitlement: model.EntitlementVIP, IsVIP: false}
	}

	isVIP := false
	if row.Status == model.EntStatusActive {
		if row.ExpiredAt == 0 || row.ExpiredAt > nowUnix {
			isVIP = true
		}
	}

	return &entdto.VIPStatusResp{
		Entitlement: model.EntitlementVIP,
		IsVIP:       isVIP,
		Status:      row.Status,
		Source:      row.Source,
		AutoRenew:   row.AutoRenew,
		StartedAt:   row.StartedAt,
		ExpiredAt:   row.ExpiredAt,
		RefID:       row.RefID,
		UpdatedAt:   row.UpdatedAt,
	}
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
