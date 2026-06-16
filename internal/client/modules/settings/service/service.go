package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"storeready_ai/internal/client/modules/settings/dto"
	"storeready_ai/internal/client/modules/settings/model"
	"storeready_ai/internal/client/modules/settings/repo"
)

// Service 用户全局设置服务（用例层）。
//
// 职责：
// - 读取/创建默认设置（GetOrCreate）
// - 校验并更新设置
// - 返回对外 DTO
//
// 说明：
// - tenant_id 必须为有效租户 ID；Service 层不再假设 MVP 固定为 0。
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
type Service struct {
	repo repo.UserSettingsRepo

	// 是否严格校验 timezone（IANA）。
	// MVP 默认开启：能在开发阶段尽早发现错误。
	strictTZ bool
}

func New(r repo.UserSettingsRepo) *Service {
	return &Service{repo: r, strictTZ: true}
}

func NewWithTZCheck(r repo.UserSettingsRepo, strict bool) *Service {
	return &Service{repo: r, strictTZ: strict}
}

// Get 获取当前用户设置（不存在则创建默认值）。
func (s *Service) Get(ctx context.Context, tenantID, userID uint64) (*dto.SettingsResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("settings service 未初始化")
	}

	row, err := s.repo.GetOrCreate(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return toResp(row), nil
}

// Update 更新当前用户设置（局部更新）。
func (s *Service) Update(ctx context.Context, tenantID, userID uint64, req *dto.UpdateSettingsReq) (*dto.SettingsResp, error) {
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("settings service 未初始化")
	}
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	row, err := s.repo.GetOrCreate(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// --- 应用更新（未传字段不改） ---
	if req.DefaultCurrency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*req.DefaultCurrency))
		if currency == "" {
			currency = model.DefaultCurrencyUSD
		}
		if len(currency) != 3 {
			return nil, errors.New("default_currency 必须为 3 位 ISO4217")
		}
		row.DefaultCurrency = currency
	}
	if req.DefaultRemindBeforeDays != nil {
		if *req.DefaultRemindBeforeDays > 30 {
			return nil, errors.New("default_remind_before_days 范围 0~30")
		}
		row.DefaultRemindBeforeDays = *req.DefaultRemindBeforeDays
	}
	if req.DefaultRemindOnDay != nil {
		if *req.DefaultRemindOnDay != 0 && *req.DefaultRemindOnDay != 1 {
			return nil, errors.New("default_remind_on_day 仅支持 0/1")
		}
		row.DefaultRemindOnDay = *req.DefaultRemindOnDay
	}
	if req.NotificationEnabled != nil {
		if *req.NotificationEnabled != 0 && *req.NotificationEnabled != 1 {
			return nil, errors.New("notification_enabled 仅支持 0/1")
		}
		row.NotificationEnabled = *req.NotificationEnabled
	}
	if req.DefaultNotifyTime != nil {
		notifyTime := strings.TrimSpace(*req.DefaultNotifyTime)
		if notifyTime != "" {
			if _, err := time.Parse("15:04:05", notifyTime); err != nil {
				return nil, errors.New("default_notify_time 格式非法，要求 HH:MM:SS")
			}
			row.DefaultNotifyTime = notifyTime
		} else {
			row.DefaultNotifyTime = model.DefaultNotifyTime0900
		}
	}
	if req.Timezone != nil {
		tz := strings.TrimSpace(*req.Timezone)
		if tz == "" {
			tz = model.DefaultTimezoneUTC
		}
		if len(tz) > 64 {
			return nil, errors.New("timezone 长度不能超过 64")
		}
		if s.strictTZ {
			if _, err := time.LoadLocation(tz); err != nil {
				return nil, errors.New("timezone 非法（需 IANA 格式，如 Asia/Shanghai）")
			}
		}
		row.Timezone = tz
	}

	// 写回数据库
	if err := s.repo.Update(ctx, row); err != nil {
		return nil, err
	}

	// Update 后再次读取（确保返回一致；也便于未来引入触发器/默认值）
	row2, err := s.repo.GetByUser(ctx, tenantID, userID)
	if err != nil {
		// 兜底：读失败就返回内存里的 row
		return toResp(row), nil
	}
	return toResp(row2), nil
}

func toResp(row *model.UserSettings) *dto.SettingsResp {
	if row == nil {
		return nil
	}
	return &dto.SettingsResp{
		ID:       row.ID,
		TenantID: row.TenantID,
		UserID:   row.UserID,

		DefaultCurrency:         row.DefaultCurrency,
		DefaultRemindBeforeDays: row.DefaultRemindBeforeDays,
		DefaultRemindOnDay:      row.DefaultRemindOnDay,
		NotificationEnabled:     row.NotificationEnabled,
		DefaultNotifyTime:       row.DefaultNotifyTime,
		Timezone:                row.Timezone,

		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}
}
