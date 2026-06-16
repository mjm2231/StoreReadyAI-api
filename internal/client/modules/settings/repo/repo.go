package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"storeready_ai/internal/client/modules/settings/model"
)

// UserSettingsRepo 用户全局设置的持久化接口（Repo 层）。
//
// 说明：
// - 与传输层无关（HTTP / 后期 gRPC 可复用同一套接口）。
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - 通过 (tenant_id,user_id) 唯一键保证每个用户只有一条设置。
// - tenant_id 必须为有效租户 ID；Repo 层不再假设 MVP 固定为 0。
type UserSettingsRepo interface {
	// GetByUser 按 tenant_id + user_id 获取设置。
	GetByUser(ctx context.Context, tenantID, userID uint64) (*model.UserSettings, error)

	// GetOrCreate 获取设置；若不存在则按默认值创建并返回。
	GetOrCreate(ctx context.Context, tenantID, userID uint64) (*model.UserSettings, error)

	// Upsert 按 (tenant_id,user_id) UPSERT。
	// - 若不存在：创建
	// - 若存在：按传入字段更新
	Upsert(ctx context.Context, row *model.UserSettings) (*model.UserSettings, error)

	// Update 更新设置（仅更新允许字段；需传入 row.ID/tenant_id/user_id）。
	Update(ctx context.Context, row *model.UserSettings) error
}

// gormRepo UserSettingsRepo 的 GORM 实现（对外隐藏具体类型，只暴露接口）。
type gormRepo struct {
	db *gorm.DB
}

// New 创建 Repo（返回接口，便于 mock / 替换实现）。
func New(db *gorm.DB) UserSettingsRepo {
	return &gormRepo{db: db}
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

func normalizeNotifyTime(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return model.DefaultNotifyTime0900, nil
	}
	if _, err := time.Parse("15:04:05", v); err != nil {
		return "", errors.New("default_notify_time 格式非法，要求 HH:MM:SS")
	}
	return v, nil
}

func (r *gormRepo) GetByUser(ctx context.Context, tenantID, userID uint64) (*model.UserSettings, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}

	var row model.UserSettings
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND deleted_at = 0", tenantID, userID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *gormRepo) GetOrCreate(ctx context.Context, tenantID, userID uint64) (*model.UserSettings, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}

	row, err := r.GetByUser(ctx, tenantID, userID)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := nowSec()
	create := &model.UserSettings{
		TenantID: tenantID,
		UserID:   userID,

		DefaultCurrency:         model.DefaultCurrencyUSD,
		DefaultRemindBeforeDays: model.DefaultRemindBeforeDays,
		DefaultRemindOnDay:      model.DefaultRemindOnDay,
		NotificationEnabled:     model.DefaultNotificationOn,
		DefaultNotifyTime:       model.DefaultNotifyTime0900,
		Timezone:                model.DefaultTimezoneUTC,

		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: 0,
	}

	// 并发下可能被别的请求先创建，使用 OnConflict DO NOTHING 保证幂等
	err = r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}}, DoNothing: true}).
		Create(create).Error
	if err != nil {
		return nil, err
	}

	// 再查一遍确保拿到真实记录（包含自增 id）
	return r.GetByUser(ctx, tenantID, userID)
}

func (r *gormRepo) Upsert(ctx context.Context, row *model.UserSettings) (*model.UserSettings, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if row == nil {
		return nil, errors.New("row 不能为空")
	}
	if row.TenantID == 0 || row.UserID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}

	// 规范化字段
	row.DefaultCurrency = strings.ToUpper(strings.TrimSpace(row.DefaultCurrency))
	if row.DefaultCurrency == "" {
		row.DefaultCurrency = model.DefaultCurrencyUSD
	}
	row.Timezone = strings.TrimSpace(row.Timezone)
	if row.Timezone == "" {
		row.Timezone = model.DefaultTimezoneUTC
	}
	notifyTime, err := normalizeNotifyTime(row.DefaultNotifyTime)
	if err != nil {
		return nil, err
	}
	row.DefaultNotifyTime = notifyTime

	now := nowSec()
	if row.CreatedAt == 0 {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	row.DeletedAt = 0

	// 以 (tenant_id,user_id) 唯一键为准做 UPSERT
	err = r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"default_currency",
				"default_remind_before_days",
				"default_remind_on_day",
				"notification_enabled",
				"default_notify_time",
				"timezone",
				"deleted_at",
				"updated_at",
			}),
		}).
		Create(row).Error
	if err != nil {
		return nil, err
	}

	return r.GetByUser(ctx, row.TenantID, row.UserID)
}

func (r *gormRepo) Update(ctx context.Context, row *model.UserSettings) error {
	if r == nil || r.db == nil {
		return errors.New("repo 未初始化")
	}
	if row == nil {
		return errors.New("row 不能为空")
	}
	if row.ID == 0 || row.TenantID == 0 || row.UserID == 0 {
		return errors.New("id/tenant_id/user_id 非法")
	}

	// 规范化字段
	currency := strings.ToUpper(strings.TrimSpace(row.DefaultCurrency))
	if currency == "" {
		currency = model.DefaultCurrencyUSD
	}
	tz := strings.TrimSpace(row.Timezone)
	if tz == "" {
		tz = model.DefaultTimezoneUTC
	}
	notifyTime, err := normalizeNotifyTime(row.DefaultNotifyTime)
	if err != nil {
		return err
	}

	now := nowSec()
	row.UpdatedAt = now

	updates := map[string]any{
		"default_currency":           currency,
		"default_remind_before_days": row.DefaultRemindBeforeDays,
		"default_remind_on_day":      row.DefaultRemindOnDay,
		"notification_enabled":       row.NotificationEnabled,
		"default_notify_time":        notifyTime,
		"timezone":                   tz,
		"updated_at":                 now,
	}

	res := r.db.WithContext(ctx).
		Model(&model.UserSettings{}).
		Where("tenant_id = ? AND user_id = ? AND id = ? AND deleted_at = 0", row.TenantID, row.UserID, row.ID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("user_settings 不存在或已删除")
	}
	return nil
}
