package model

// UserDevice 对应表 `user_devices`（用户设备表：同步/多端）。
//
// 约定：
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - tenant_id 在 MVP 固定为 0，但仍保留字段用于未来多租户扩展。
// - device_id 由客户端生成并持久化（同一用户 + device_id 唯一）。
//
// 平台：platform
// - 1=iOS, 2=Android, 3=Web, 9=Unknown
//
// 状态：status
// - 1=active, 2=revoked
//
// 注意：
// - UNIQUE 约束 uk_user_device(tenant_id,user_id,device_id) 建议通过 SQL migration 创建；gorm tag 仅声明索引名。
// - 索引 idx_device_seen/idx_device_status 同上。
type UserDevice struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement;column:id"` // 主键
	TenantID uint64 `gorm:"not null;default:0;column:tenant_id;uniqueIndex:uk_user_device,priority:1;index:idx_device_seen,priority:1;index:idx_device_status,priority:1"`
	UserID   uint64 `gorm:"not null;column:user_id;uniqueIndex:uk_user_device,priority:2;index:idx_device_seen,priority:2;index:idx_device_status,priority:2"`

	DeviceID   string  `gorm:"type:varchar(128);not null;column:device_id;uniqueIndex:uk_user_device,priority:3"` // 设备唯一ID
	Platform   uint8   `gorm:"type:tinyint;not null;column:platform"`                                             // 平台
	DeviceName *string `gorm:"type:varchar(128);column:device_name"`                                              // 设备名(可选)
	AppVersion *string `gorm:"type:varchar(32);column:app_version"`                                               // App版本(可选)

	PushToken *string `gorm:"type:varchar(256);column:push_token"` // 推送Token(APNs/FCM，可选)
	LastIP    *string `gorm:"type:varchar(64);column:last_ip"`     // 最近IP(可选)
	UserAgent *string `gorm:"type:varchar(256);column:user_agent"` // User-Agent(可选)

	Status uint8 `gorm:"type:tinyint;not null;default:1;column:status;index:idx_device_status,priority:3"` // 状态:1=active,2=revoked

	LastSeenAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:last_seen_at;index:idx_device_seen,priority:3"` // 最近活跃时间
	LastSyncAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:last_sync_at"`                                  // 最近同步时间

	CreatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:created_at"`
	UpdatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:updated_at"`
}

func (UserDevice) TableName() string { return "user_devices" }

// Platform 枚举
const (
	PlatformIOS     uint8 = 1
	PlatformAndroid uint8 = 2
	PlatformWeb     uint8 = 3
	PlatformUnknown uint8 = 9
)

// DeviceStatus 枚举
const (
	DeviceStatusActive  uint8 = 1
	DeviceStatusRevoked uint8 = 2
)
