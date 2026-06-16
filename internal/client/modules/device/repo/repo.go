package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"storeready_ai/internal/client/modules/device/model"
)

// DeviceRepo 用户设备的持久化接口（Repo 层）。
//
// 说明：
// - 与传输层无关（HTTP / 后期 gRPC 可复用同一套接口）。
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - 通过 (tenant_id,user_id,device_id) 唯一键保证同一用户同一设备只会有一条记录。
//
// MVP：tenant_id 固定为 0，但接口仍保留字段便于未来扩展。
type DeviceRepo interface {
	// UpsertDevice 设备登记/更新（幂等）。
	// 使用 uk_user_device(tenant_id,user_id,device_id) 做 UPSERT。
	UpsertDevice(ctx context.Context, row *model.UserDevice) (*model.UserDevice, error)

	// TouchSeen 更新设备最近活跃时间（last_seen_at），可顺带更新 push_token/app_version 等。
	TouchSeen(ctx context.Context, tenantID, userID uint64, deviceID string, patch TouchPatch) error

	// TouchSync 更新设备最近同步时间（last_sync_at）。
	TouchSync(ctx context.Context, tenantID, userID uint64, deviceID string, lastSyncAt uint64) error

	// ListActiveDevices 列出当前用户的 active 设备。
	ListActiveDevices(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.UserDevice, int64, error)

	// RevokeDevice 将设备标记为 revoked（软踢设备，不删除记录）。
	RevokeDevice(ctx context.Context, tenantID, userID uint64, deviceID string) error
}

// TouchPatch TouchSeen 时可更新的可选字段。
type TouchPatch struct {
	Platform   *uint8
	DeviceName *string
	AppVersion *string
	PushToken  *string
	LastIP     *string
	UserAgent  *string
}

// gormRepo DeviceRepo 的 GORM 实现（对外隐藏具体类型，只暴露接口）。
type gormRepo struct {
	db *gorm.DB
}

// New 创建 Repo（返回接口，便于 mock / 替换实现）。
func New(db *gorm.DB) DeviceRepo {
	return &gormRepo{db: db}
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

func (r *gormRepo) UpsertDevice(ctx context.Context, row *model.UserDevice) (*model.UserDevice, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if row == nil {
		return nil, errors.New("row 不能为空")
	}
	if row.TenantID == 0 || row.UserID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	row.DeviceID = strings.TrimSpace(row.DeviceID)
	if row.DeviceID == "" {
		return nil, errors.New("device_id 不能为空")
	}
	if row.Platform == 0 {
		row.Platform = model.PlatformUnknown
	}
	if row.Status == 0 {
		row.Status = model.DeviceStatusActive
	}

	now := nowSec()
	if row.CreatedAt == 0 {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	if row.LastSeenAt == 0 {
		row.LastSeenAt = now
	}

	// UPSERT：存在则更新可变字段
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}, {Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"platform",
				"device_name",
				"app_version",
				"push_token",
				"last_ip",
				"user_agent",
				"status",
				"last_seen_at",
				"updated_at",
			}),
		}).
		Create(row).Error
	if err != nil {
		return nil, err
	}

	// 尽量返回带 ID 的记录
	if row.ID != 0 {
		return row, nil
	}

	var out model.UserDevice
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND device_id = ?", row.TenantID, row.UserID, row.DeviceID).
		First(&out).Error; err != nil {
		return row, nil
	}
	return &out, nil
}

func (r *gormRepo) TouchSeen(ctx context.Context, tenantID, userID uint64, deviceID string, patch TouchPatch) error {
	if r == nil || r.db == nil {
		return errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}

	now := nowSec()
	updates := map[string]any{
		"last_seen_at": now,
		"updated_at":   now,
		"status":       model.DeviceStatusActive,
	}

	if patch.Platform != nil && *patch.Platform != 0 {
		updates["platform"] = *patch.Platform
	}
	if patch.DeviceName != nil {
		updates["device_name"] = strings.TrimSpace(*patch.DeviceName)
	}
	if patch.AppVersion != nil {
		updates["app_version"] = strings.TrimSpace(*patch.AppVersion)
	}
	if patch.PushToken != nil {
		v := strings.TrimSpace(*patch.PushToken)
		if v == "" {
			updates["push_token"] = nil
		} else {
			updates["push_token"] = v
		}
	}
	if patch.LastIP != nil {
		v := strings.TrimSpace(*patch.LastIP)
		if v == "" {
			updates["last_ip"] = nil
		} else {
			updates["last_ip"] = v
		}
	}
	if patch.UserAgent != nil {
		v := strings.TrimSpace(*patch.UserAgent)
		if v == "" {
			updates["user_agent"] = nil
		} else {
			updates["user_agent"] = v
		}
	}

	res := r.db.WithContext(ctx).
		Model(&model.UserDevice{}).
		Where("tenant_id = ? AND user_id = ? AND device_id = ?", tenantID, userID, deviceID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}

	// 如果没更新到（未登记），则插入一条
	if res.RowsAffected == 0 {
		row := &model.UserDevice{
			TenantID: tenantID,
			UserID:   userID,
			DeviceID: deviceID,
			Platform: model.PlatformUnknown,

			Status: model.DeviceStatusActive,

			LastSeenAt: now,
			LastSyncAt: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if patch.Platform != nil && *patch.Platform != 0 {
			row.Platform = *patch.Platform
		}
		if patch.DeviceName != nil {
			v := strings.TrimSpace(*patch.DeviceName)
			if v != "" {
				row.DeviceName = &v
			}
		}
		if patch.AppVersion != nil {
			v := strings.TrimSpace(*patch.AppVersion)
			if v != "" {
				row.AppVersion = &v
			}
		}
		if patch.PushToken != nil {
			v := strings.TrimSpace(*patch.PushToken)
			if v != "" {
				row.PushToken = &v
			}
		}
		if patch.LastIP != nil {
			v := strings.TrimSpace(*patch.LastIP)
			if v != "" {
				row.LastIP = &v
			}
		}
		if patch.UserAgent != nil {
			v := strings.TrimSpace(*patch.UserAgent)
			if v != "" {
				row.UserAgent = &v
			}
		}

		_, err := r.UpsertDevice(ctx, row)
		return err
	}

	return nil
}

func (r *gormRepo) TouchSync(ctx context.Context, tenantID, userID uint64, deviceID string, lastSyncAt uint64) error {
	if r == nil || r.db == nil {
		return errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}
	if lastSyncAt == 0 {
		lastSyncAt = nowSec()
	}

	updates := map[string]any{
		"last_sync_at": lastSyncAt,
		"updated_at":   nowSec(),
	}

	res := r.db.WithContext(ctx).
		Model(&model.UserDevice{}).
		Where("tenant_id = ? AND user_id = ? AND device_id = ?", tenantID, userID, deviceID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}

	// 未登记则创建
	if res.RowsAffected == 0 {
		now := nowSec()
		row := &model.UserDevice{
			TenantID: tenantID,
			UserID:   userID,
			DeviceID: deviceID,
			Platform: model.PlatformUnknown,
			Status:   model.DeviceStatusActive,

			LastSeenAt: now,
			LastSyncAt: lastSyncAt,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		_, err := r.UpsertDevice(ctx, row)
		return err
	}

	return nil
}

func (r *gormRepo) ListActiveDevices(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.UserDevice, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return nil, 0, errors.New("tenant_id/user_id 非法")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	q := r.db.WithContext(ctx).Model(&model.UserDevice{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Where("status = ?", model.DeviceStatusActive)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.UserDevice{}, 0, nil
	}

	var rows []*model.UserDevice
	if err := q.
		Order("last_seen_at desc, id desc").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *gormRepo) RevokeDevice(ctx context.Context, tenantID, userID uint64, deviceID string) error {
	if r == nil || r.db == nil {
		return errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return errors.New("tenant_id/user_id 非法")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device_id 不能为空")
	}

	now := nowSec()
	return r.db.WithContext(ctx).
		Model(&model.UserDevice{}).
		Where("tenant_id = ? AND user_id = ? AND device_id = ?", tenantID, userID, deviceID).
		Updates(map[string]any{"status": model.DeviceStatusRevoked, "updated_at": now}).Error
}
