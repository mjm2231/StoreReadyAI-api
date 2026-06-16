package model

// UserSettings 对应表 `user_settings`（用户全局设置）。
//
// 约定：
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - tenant_id 在 MVP 固定为 0，但仍保留字段用于未来多租户扩展。
// - timezone 使用 IANA 时区（如 Asia/Shanghai）。
//
// 注意：
// - UNIQUE 约束（tenant_id,user_id）建议通过 SQL migration 创建；这里的 gorm tag 仅用于声明索引名。
// - 如你使用 AutoMigrate，可能不会完全创建/同步所有索引细节，仍以 SQL migration 为准。
//
// 字段对齐 SQL：
// - default_currency: CHAR(3)
// - default_remind_before_days: SMALLINT UNSIGNED (0~30)
// - default_remind_on_day / notification_enabled: TINYINT (0/1)
// - default_notify_time: TIME
// - timezone: VARCHAR(64)
// - created_at/updated_at/deleted_at: BIGINT UNSIGNED (Unix 秒)
type UserSettings struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement;column:id"` // 主键
	TenantID uint64 `gorm:"not null;default:0;column:tenant_id;uniqueIndex:uk_settings_user,priority:1;index:idx_settings_sync,priority:1"`
	UserID   uint64 `gorm:"not null;column:user_id;uniqueIndex:uk_settings_user,priority:2"`

	DefaultCurrency         string `gorm:"type:char(3);not null;default:'USD';column:default_currency"`                 // 默认币种
	DefaultRemindBeforeDays uint16 `gorm:"type:smallint unsigned;not null;default:3;column:default_remind_before_days"` // 默认提前提醒天数(0~30)
	DefaultRemindOnDay      uint8  `gorm:"type:tinyint;not null;default:1;column:default_remind_on_day"`                // 默认到期当天提醒(0/1)
	NotificationEnabled     uint8  `gorm:"type:tinyint;not null;default:1;column:notification_enabled"`                 // 通知总开关(0/1)
	DefaultNotifyTime       string `gorm:"type:time;not null;default:'09:00:00';column:default_notify_time"`            // 默认通知时间(HH:MM:SS)
	Timezone                string `gorm:"type:varchar(64);not null;default:'UTC';column:timezone"`                     // 时区(IANA)

	CreatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:created_at"`
	UpdatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:updated_at;index:idx_settings_sync,priority:2"`
	DeletedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:deleted_at"`
}

func (UserSettings) TableName() string { return "user_settings" }

// 默认值常量（与 SQL 默认值对齐）
const (
	DefaultCurrencyUSD    = "USD"
	DefaultTimezoneUTC    = "UTC"
	DefaultNotifyTime0900 = "09:00:00"

	DefaultRemindBeforeDays uint16 = 3
	DefaultRemindOnDay      uint8  = 1
	DefaultNotificationOn   uint8  = 1
)
