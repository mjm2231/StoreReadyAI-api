package service

import (
	"context"
	"errors"
	"strings"
	"time"

	devicedto "storeready_ai/internal/client/modules/device/dto"
	"storeready_ai/internal/client/modules/device/model"
	devicerepo "storeready_ai/internal/client/modules/device/repo"
)

// Service 设备服务（用例层）。
//
// 职责（MVP）：
// - 设备登记（register/upsert）
// - 心跳（更新 last_seen_at，并可 patch push_token/app_version 等）
// - 同步时间点上报（更新 last_sync_at）
// - 设备列表（仅 active）
// - 撤销设备（revoked）
//
// 说明：
// - user_id 由上层 handler 从 common.GetUID 获取并传入。
// - last_ip/user_agent 推荐由 handler 从请求提取并传入 patch；service 不直接依赖 HTTP。
type Service struct {
	repo devicerepo.DeviceRepo
}

func New(r devicerepo.DeviceRepo) *Service {
	return &Service{repo: r}
}

// Register 设备登记（幂等）。
func (s *Service) Register(ctx context.Context, tenantID, userID uint64, req *devicedto.RegisterDeviceReq, lastIP, userAgent *string) (*devicedto.DeviceItem, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("device service 未初始化")
	}
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, errors.New("device_id 不能为空")
	}

	now := uint64(time.Now().Unix())
	row := &model.UserDevice{
		TenantID: tenantID,
		UserID:   userID,

		DeviceID:   deviceID,
		Platform:   normalizePlatform(req.Platform),
		DeviceName: trimPtr(req.DeviceName),
		AppVersion: trimPtr(req.AppVersion),
		PushToken:  trimPtr(req.PushToken),
		LastIP:     trimPtr(lastIP),
		UserAgent:  trimPtr(userAgent),

		Status: model.DeviceStatusActive,

		LastSeenAt: now,
		// last_sync_at 不在 register 强制更新，由客户端完成 sync 后上报
		LastSyncAt: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	saved, err := s.repo.UpsertDevice(ctx, row)
	if err != nil {
		return nil, err
	}
	return toItem(saved), nil
}

// Heartbeat 心跳/活跃上报。
func (s *Service) Heartbeat(ctx context.Context, tenantID, userID uint64, req *devicedto.HeartbeatReq, lastIP, userAgent *string) error {
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return errors.New("device service 未初始化")
	}
	if req == nil {
		return errors.New("请求不能为空")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}

	patch := devicerepo.TouchPatch{
		Platform:   req.Platform,
		DeviceName: req.DeviceName,
		AppVersion: req.AppVersion,
		PushToken:  req.PushToken,
		LastIP:     lastIP,
		UserAgent:  userAgent,
	}

	return s.repo.TouchSeen(ctx, tenantID, userID, deviceID, patch)
}

// TouchSync 同步时间点上报（客户端完成一次 pull/push 后调用）。
func (s *Service) TouchSync(ctx context.Context, tenantID, userID uint64, req *devicedto.TouchSyncReq) error {
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return errors.New("device service 未初始化")
	}
	if req == nil {
		return errors.New("请求不能为空")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}
	return s.repo.TouchSync(ctx, tenantID, userID, deviceID, req.LastSyncAt)
}

// ListActiveDevices 查询当前用户的 active 设备。
func (s *Service) ListActiveDevices(ctx context.Context, tenantID, userID uint64, req *devicedto.ListDevicesReq) (*devicedto.ListDevicesResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("device service 未初始化")
	}
	if req == nil {
		req = &devicedto.ListDevicesReq{}
	}

	items, total, err := s.repo.ListActiveDevices(ctx, tenantID, userID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	resp := &devicedto.ListDevicesResp{Total: total, Items: make([]*devicedto.DeviceItem, 0, len(items))}
	for _, it := range items {
		if it == nil {
			continue
		}
		resp.Items = append(resp.Items, toItem(it))
	}
	return resp, nil
}

// Revoke 撤销设备（踢设备）。
func (s *Service) Revoke(ctx context.Context, tenantID, userID uint64, req *devicedto.RevokeDeviceReq) error {
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return errors.New("device service 未初始化")
	}
	if req == nil {
		return errors.New("请求不能为空")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}
	return s.repo.RevokeDevice(ctx, tenantID, userID, deviceID)
}

// --- mapper ---

func toItem(row *model.UserDevice) *devicedto.DeviceItem {
	if row == nil {
		return nil
	}
	return &devicedto.DeviceItem{
		ID:       row.ID,
		TenantID: row.TenantID,
		UserID:   row.UserID,

		DeviceID:   row.DeviceID,
		Platform:   row.Platform,
		DeviceName: row.DeviceName,
		AppVersion: row.AppVersion,
		PushToken:  row.PushToken,

		Status: row.Status,

		LastSeenAt: row.LastSeenAt,
		LastSyncAt: row.LastSyncAt,

		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
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

func normalizePlatform(p uint8) uint8 {
	switch p {
	case model.PlatformIOS, model.PlatformAndroid, model.PlatformWeb, model.PlatformUnknown:
		return p
	default:
		return model.PlatformUnknown
	}
}
